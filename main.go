package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-sat"
	"github.com/mitchellh/go-sat/cnf"
	"gonum.org/v1/gonum/stat/combin"
)

const inputFormatMsg = "Please input n^2 * n^2 numbers 0 or 1-9 delimitted by conma. 0 is empty as Sudoku cell."

// sudokuは実質的な主処理である.
func sudoku() int {
	flag.Usage = flagUsage
	flag.Parse()

	// 引数の有無を検証する.
	args := flag.Args()
	if len(args) != 1 {
		flagUsage()
		return 1
	}

	// 入力ファイルをパースし, 形式を検証する.
	input, err := parseProblem(args[0])
	if err != nil {
		printError(err)
		return 1
	}

	// パースされた問題を表示する.
	fmt.Println("[Input problem]")
	for _, row := range input {
		for _, n := range row {
			fmt.Printf("%2d ", n)
		}
		fmt.Println()
	}
	fmt.Println()

	// 以降を処理時間の計測対象とする.
	st := time.Now()

	// 問題のサイズは複数回使うため, 変数にする.
	// sqPow2は行と列のそれぞれの長さかつセルに入る数字の最大値, sqはその平方数, sqPow4はsqの4乗である.
	sq := int(math.Sqrt(float64(len(input))))
	sqPow2 := int(math.Pow(float64(sq), float64(2)))
	sqPow4 := int(math.Pow(float64(sq), float64(4)))

	// 符号化した問題を代入するスライスを宣言する.
	// 各スライスは最終的にCNFの節となる.
	var encSlices [][]int

	/*
		問題を3次元 x[i][j][k] で捉え符号化する.
		iは行の, jは列のインデックス, kはセルに入る数字とする.
		例えば行番号1, 列番号1のセルに1が入る場合の符号は1に, 4では4になる.
		x[1][1][1] = 1
		x[1][1][4] = 4
		1<=i,j,k<=sqPow2であり, 行と列のインデックス増加時にはsqPow2を必要数足して符号化する.
		例えば行番号1, 列番号2のセルに1が入り, sqPow2が4の場合の符号は5に, セルに4が入る場合は8になる.
		x[1][2][1] = 5
		x[1][2][4] = 8
		なお, 各ループ処理のインデックスとは異なる.
	*/

	/*
		各セルのとりうる値を論理式で表現し, スライスにまとめる.
		例えば4x4の場合は, 各セルには1から4の数字が入る.
		セル(1,1)の論理式は (1 v 2 v 3 v 4) で, スライスは [1 2 3 4] となる.
		インデックスiは行, jは列, kはセルに入る数字である.
	*/
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			var r []int
			for k := 1; k <= sqPow2; k++ {
				r = append(r, (sqPow4*(i-1))+(sqPow2*(j-1))+k)
			}
			encSlices = append(encSlices, r)
		}
	}

	/*
		各行のとりうる値を論理式で表現し, スライスにまとめる.
		例えば各行のいずれかのセルに1が入る.
		4x4とすると, その場合の行1の論理式は (1 v 5 v 9 v 13) で, スライスは [1 5 9 13] となる.
		インデックスiはセルに入る数字,jは行,kは列である.
	*/
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			var r []int
			for k := 1; k <= sqPow2; k++ {
				r = append(r, i+(sqPow4*(j-1))+(sqPow2*(k-1)))
			}
			encSlices = append(encSlices, r)
		}
	}

	/*
		各列のとりうる値を論理式で表現し, スライスにまとめる.
		例えば各列のいずれかのセルに1が入る.
		4x4とすると, その場合の列1の論理式は (1 v 17 v 33 v 49) なので, スライスは [1 17 33 49] となる.
		インデックスiはセルに入る数字, jは列, kは行である.
	*/
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			var r []int
			for k := 1; k <= sqPow2; k++ {
				r = append(r, i+(sqPow2*(j-1))+(sqPow4*(k-1)))
			}
			encSlices = append(encSlices, r)
		}
	}

	/*
		/ブロックのとりうる値を論理式で表現し, スライスにまとめる. ブロックの行と列の長さはsqである.
		例えば各ブロックのいずれかのセルに1が入る.
		4x4とすると, その場合のブロック1の論理式は (1 v 5 v 17 v 21) なので, スライスは [1 17 33 49] となる.
		インデックスiはセルに入る数字, jはブロック, kはブロック内の位置である.
	*/
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			var r []int
			for k := 1; k <= sqPow2; k++ {
				r = append(r, i+
					((j-1)%sq)*sq*sqPow2+ /* ブロックに応じて列方向の加算 */
					int((j-1)/sq)*sq*sqPow4+ /* ブロックに応じて行方向の加算 */
					((k-1)%sq)*sqPow2+ /* ブロック内の位置に応じて列方向の加算 */
					int((k-1)/sq)*(sqPow2*sqPow2), /* ブロック内の位置に応じて行方向の加算 */
				)
			}
			encSlices = append(encSlices, r)
		}
	}

	// CNF形式の問題を代入するスライスを宣言する.
	var cnfSlices [][]int

	// 先に符号化した「少なくとも1つが真」となるスライスを全て代入する.
	for _, r := range encSlices {
		cnfSlices = append(cnfSlices, r)
	}

	// 「少なくとも1つが真」となるスライスを元に,「たかだか1つが真」を表現するスライスを作り代入する.
	for _, r := range encSlices {
		c := combinations(r)
		for _, s := range c {
			cnfSlices = append(cnfSlices, s)
		}
	}

	// 入力された問題を節として追加する.
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			if input[i-1][j-1] != 0 {
				s := []int{(i-1)*sqPow4 + (j-1)*sqPow2 + input[i-1][j-1]}
				cnfSlices = append(cnfSlices, s)
			}
		}
	}

	// CNFの大きさ(節数)を表示する.
	fmt.Printf("Number of generated CNF clauses: %v\n", len(cnfSlices))

	// 充足可否と付値を取得する.
	// SATソルバーにはMITライセンスで公開されている go-sat を利用する(https://github.com/mitchellh/go-sat)
	formula := cnf.NewFormulaFromInts(cnfSlices)
	s := sat.New()
	s.AddFormula(formula)
	r := s.Solve()
	fmt.Printf("SAT: %v\n", r)
	fmt.Println()
	if !r {
		return 0
	}
	as := s.Assignments()

	// 真の要素を選び, ソートする.
	keys := []int{}
	for k, a := range as {
		if a {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	// 要素を復号して表示する.
	fmt.Println("[Solution]")
	sol := []int{}
	for k, a := range keys {
		sol = append(sol, a-(k*sqPow2))
		if (k+1)%sqPow2 == 0 {
			for _, n := range sol {
				fmt.Printf("%2d ", n)
			}
			fmt.Println()
			sol = []int{}
		}
	}
	fmt.Println()

	// 処理時間を表示する.
	et := time.Now()
	fmt.Println("Time: ", et.Sub(st))

	return 0
}

// parseProblemは数独の問題ファイルを受け取り, 形式を検証する.
func parseProblem(fn /* filename */ string) ([][]int, error) {
	re := regexp.MustCompile("[0-9]+")
	var input [][]int

	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		l := scanner.Text()
		r := csv.NewReader(strings.NewReader(l))
		nums, err := r.Read()
		if err != nil {
			return nil, err
		}

		var s []int
		for _, num := range nums {
			if !re.MatchString(num) {
				return nil, fmt.Errorf(inputFormatMsg)
			}
			n, _ := strconv.Atoi(num)
			s = append(s, n)
		}
		input = append(input, s)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	columnCount := 0
	rowCount := len(input)
	for _, row := range input {
		if len(row) == 0 {
			return nil, fmt.Errorf(inputFormatMsg)
		}
		if columnCount == 0 || columnCount == len(row) {
			columnCount = len(row)
		} else {
			return nil, fmt.Errorf(inputFormatMsg)
		}
	}
	if columnCount != rowCount {
		return nil, fmt.Errorf(inputFormatMsg)
	}
	sq := int(math.Sqrt(float64(columnCount)))
	if sq*sq != columnCount {
		return nil, fmt.Errorf(inputFormatMsg)
	}

	return input, nil
}

// combinationsはスライス要素の組み合わせ(nC2)を作り, かつ否定の選言を表現するため各要素を負数に変換する.
func combinations(s /* slice */ []int) [][]int {
	var r [][]int
	cs := combin.Combinations(len(s), 2)
	for _, c := range cs {
		t := []int{-s[c[0]], -s[c[1]]}
		r = append(r, t)
	}
	return r
}

// flagUsageはコマンドラインオプション(フラグ)の使い方を出力する.
func flagUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %[1]s <problem-filename>\n", os.Args[0])
	flag.PrintDefaults()
}

// printErrorはエラーメッセージ出力を統一する.
func printError(err error) {
	fmt.Fprintf(os.Stderr, err.Error()+"\n")
}

// mainはエントリーポイントと終了コードを返却する役割のみとする.
func main() {
	os.Exit(sudoku())
}
