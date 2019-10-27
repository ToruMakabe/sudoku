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

func sudoku() int {
	st := time.Now()

	flag.Usage = flagUsage
	flag.Parse()

	// 引数の有無を検証.
	args := flag.Args()
	if len(args) != 1 {
		flagUsage()
		return 1
	}

	// 入力ファイルをパースし,形式を検証する.
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

	// 問題のサイズは複数回使うため,変数にする.
	// sqPow2は行と列のそれぞれの長さかつセルに入る数字の最大値,sqはその平方数,sqPow4はsqの4乗である.
	sq := int(math.Sqrt(float64(len(input))))
	sqPow2 := int(math.Pow(float64(sq), float64(2)))
	sqPow4 := int(math.Pow(float64(sq), float64(4)))

	// 符号化した問題を代入するスライスを宣言する.
	// 各スライスは最終的にCNFの節となる.
	var encSlices [][]int

	/*
		問題を3次元 x[i][j][k] で捉え符号化する.
		iは行,jは列,kはセルのインデックスとする.
		例えば行番号1,列番号1のセルに1が入る場合の符号は1に,4では4になる.
		x[1][1][1] = 1
		x[1][1][4] = 4
		1<=i,j,k<=sqPow2であり,行と列のインデックス増加時にはsqPow2を必要数足して符号化する.
		例えば行番号1,列番号2のセルに1が入り,sqPow2が4の場合の符号は5に,5では8になる.
		x[1][2][1] = 5
		x[1][2][4] = 8
	*/

	// 各セルのとりうる値を論理式で表現し,スライスにまとめる.
	// 例えば x[1][1][k] で 1<=k<=4 の場合の論理式は (1 v 2 v 3 v 4) なので,スライスは [1 2 3 4] となる.
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			r := []int{}
			for k := 1; k <= sqPow2; k++ {
				r = append(r, (sqPow4*(i-1))+(sqPow2*(j-1))+k)
			}
			encSlices = append(encSlices, r)
		}
	}

	// 各行のとりうる値を論理式で表現し,スライスにまとめる.
	// 例えば x[1][k][1] で 1<=k<=4 の場合の論理式は (1 v 5 v 9 v 13) なので,スライスは [1 5 9 13] となる.
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			r := []int{}
			for k := 1; k <= sqPow2; k++ {
				r = append(r, i+(sqPow4*(j-1))+(sqPow2*(k-1)))
			}
			encSlices = append(encSlices, r)
		}
	}

	// 各列のとりうる値を論理式で表現し,スライスにまとめる.
	// 例えば x[k][1][1] で 1<=k<=4 の場合の論理式は (1 v 17 v 33 v 49) なので,スライスは [1 17 33 49] となる.
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			r := []int{}
			for k := 1; k <= sqPow2; k++ {
				r = append(r, i+(sqPow2*(j-1))+(sqPow4*(k-1)))
			}
			encSlices = append(encSlices, r)
		}
	}

	// 各ブロックのとりうる値を論理式で表現し,スライスにまとめる.ブロックの行と列の長さはsqである.
	// [要追加]
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			r := []int{}
			for k := 1; k <= sqPow2; k++ {
				r = append(r, i+
					((j-1)%sq)*sq*sqPow2+ /* ブロック数に応じて列方向の加算 */
					int((j-1)/sq)*sq*sqPow4+ /* ブロック数に応じて行方向の加算 */
					((k-1)%sq)*sqPow2+ /* ブロック内で列方向の加算 */
					int((k-1)/sq)*(sqPow2*sqPow2), /* ブロック内で行方向の加算 */
				)
			}
			encSlices = append(encSlices, r)
		}
	}

	// CNFを代入するスライスを宣言.
	var cnfSlices [][]int

	// at least one.
	for _, r := range encSlices {
		cnfSlices = append(cnfSlices, r)
	}

	// at most one.
	for _, r := range encSlices {
		c := combinations(r)
		for _, s := range c {
			cnfSlices = append(cnfSlices, s)
		}
	}

	// 問題節を追加.
	for i := 1; i <= sqPow2; i++ {
		for j := 1; j <= sqPow2; j++ {
			if input[i-1][j-1] != 0 {
				s := []int{(i-1)*sqPow4 + (j-1)*sqPow2 + input[i-1][j-1]}
				cnfSlices = append(cnfSlices, s)
			}
		}
	}

	// CNFの大きさ(節数)を表示
	fmt.Printf("Number of generated CNF clauses: %v\n", len(cnfSlices))

	// go-satで充足可否と付値を取得
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

	// 付値がソートされていないためソートする.
	keys := []int{}
	for k, a := range as {
		if a {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	// 解を復号して表示.
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

	// 処理時間を表示.
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

		s := []int{}
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

	rowLength := 0
	columnLength := len(input)
	for _, row := range input {
		if len(row) == 0 {
			return nil, fmt.Errorf(inputFormatMsg)
		}
		if rowLength == 0 || rowLength == len(row) {
			rowLength = len(row)
		} else {
			return nil, fmt.Errorf(inputFormatMsg)
		}
	}
	if rowLength != columnLength {
		return nil, fmt.Errorf(inputFormatMsg)
	}
	sq := int(math.Sqrt(float64(rowLength)))
	if sq*sq != rowLength {
		return nil, fmt.Errorf(inputFormatMsg)
	}

	return input, nil
}

// combinationsはスライス要素の組み合わせ(nC2)を作り, 各要素を負数に変換する.
func combinations(s /* slice */ []int) [][]int {
	var r [][]int
	cs := combin.Combinations(len(s), 2)
	for _, c := range cs {
		t := []int{-s[c[0]], -s[c[1]]}
		r = append(r, t)
	}
	return r
}

func flagUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %[1]s <problem-filename>\n", os.Args[0])
	flag.PrintDefaults()
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, err.Error()+"\n")
}

func main() {
	os.Exit(sudoku())
}
