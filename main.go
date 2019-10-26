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

func main() {
	st := time.Now()
	re := regexp.MustCompile("[0-9]+")
	const inputFormatMsg = "Please input n * n numbers [0-n] delimitted by conma. n must be a square number such as 4. 0 is empty as Sudoku cell."

	datafilePtr := flag.String("data", "", "filepath of input data")
	flag.Parse()

	file, err := os.Open(*datafilePtr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	input := [][]int{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		r := csv.NewReader(strings.NewReader(line))
		nums, err := r.Read()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		s := []int{}
		for _, num := range nums {
			if !re.MatchString(num) {
				fmt.Println(inputFormatMsg)
				os.Exit(1)
			}
			n, _ := strconv.Atoi(num)
			s = append(s, n)
		}
		input = append(input, s)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rowLength := 0
	columnLength := len(input)
	for _, row := range input {
		if len(row) == 0 {
			fmt.Println(inputFormatMsg)
			os.Exit(1)
		}
		if rowLength == 0 || rowLength == len(row) {
			rowLength = len(row)
		} else {
			fmt.Println(inputFormatMsg)
			os.Exit(1)
		}
	}
	if rowLength != columnLength {
		fmt.Println(inputFormatMsg)
		os.Exit(1)
	}
	base := int(math.Sqrt(float64(rowLength)))
	if base*base != rowLength {
		fmt.Println(inputFormatMsg)
		os.Exit(1)
	}

	fmt.Println("[Input problem]")
	for _, row := range input {
		for _, n := range row {
			fmt.Printf("%2d ", n)
		}
		fmt.Println()
	}
	fmt.Println()

	basePow2 := int(math.Pow(float64(base), float64(2)))
	basePow4 := int(math.Pow(float64(base), float64(4)))

	encSlices := [][]int{}

	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, (basePow4*(i-1))+(basePow2*(j-1))+k)
			}
			encSlices = append(encSlices, r)
		}
	}

	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, i+(basePow4*(j-1))+(basePow2*(k-1)))
			}
			encSlices = append(encSlices, r)
		}
	}

	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, i+(basePow2*(j-1))+(basePow4*(k-1)))
			}
			encSlices = append(encSlices, r)
		}
	}

	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, i+((j-1)%base)*base*basePow2+int((j-1)/base)*base*basePow4+((k-1)%base)*basePow2+int((k-1)/base)*(basePow2*basePow2))
			}
			encSlices = append(encSlices, r)
		}
	}

	cnfSlices := [][]int{}

	for _, r := range encSlices {
		cnfSlices = append(cnfSlices, r)
	}
	for _, r := range encSlices {
		c := combinations(r)
		for _, s := range c {
			cnfSlices = append(cnfSlices, s)
		}
	}

	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			if input[i-1][j-1] != 0 {
				s := []int{(i-1)*basePow4 + (j-1)*basePow2 + input[i-1][j-1]}
				cnfSlices = append(cnfSlices, s)
			}
		}
	}

	fmt.Printf("Number of generated CNF clauses: %v\n", len(cnfSlices))

	formula := cnf.NewFormulaFromInts(cnfSlices)
	s := sat.New()
	s.AddFormula(formula)
	r := s.Solve()
	fmt.Printf("SAT: %v\n", r)
	fmt.Println()
	if !r {
		os.Exit(0)
	}
	as := s.Assignments()
	keys := []int{}
	for k, a := range as {
		if a {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	fmt.Println("[Solution]")
	sol := []int{}
	for k, a := range keys {
		sol = append(sol, a-(k*basePow2))
		if (k+1)%basePow2 == 0 {
			for _, n := range sol {
				fmt.Printf("%2d ", n)
			}
			fmt.Println()
			sol = []int{}
		}
	}
	fmt.Println()

	et := time.Now()
	fmt.Println("Time: ", et.Sub(st))

}

func combinations(s []int) [][]int {
	r := [][]int{}
	cs := combin.Combinations(len(s), 2)
	for _, c := range cs {
		t := []int{-s[c[0]], -s[c[1]]}
		r = append(r, t)
	}
	return r
}
