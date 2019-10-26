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

	"github.com/mitchellh/go-sat"
	"github.com/mitchellh/go-sat/cnf"
	"gonum.org/v1/gonum/stat/combin"
)

func main() {
	re := regexp.MustCompile("[0-9]")
	const inputFormatMsg = "Please input n * n numbers [0-n] delimitted by conma. 0 is empty as Sudoku cell."

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

	rowSize := 0
	columnSize := len(input)
	for _, row := range input {
		if len(row) == 0 {
			fmt.Println(inputFormatMsg)
			os.Exit(1)
		}
		if rowSize == 0 || rowSize == len(row) {
			rowSize = len(row)
		} else {
			fmt.Println(inputFormatMsg)
			os.Exit(1)
		}
	}
	if rowSize != columnSize {
		fmt.Println(inputFormatMsg)
		os.Exit(1)
	}

	fmt.Println("Input problem is")
	for _, row := range input {
		fmt.Println(row)
	}

	base := int(math.Sqrt(float64(rowSize)))
	basePow2 := int(math.Pow(float64(base), float64(2)))
	basePow4 := int(math.Pow(float64(base), float64(4)))

	cnfInt := [][]int{}

	rule1 := [][]int{}
	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, (basePow4*(i-1))+(basePow2*(j-1))+k)
			}
			rule1 = append(rule1, r)
		}
	}
	for _, r := range rule1 {
		cnfInt = append(cnfInt, r)
	}
	for _, r := range rule1 {
		c := combinations(r)
		for _, s := range c {
			cnfInt = append(cnfInt, s)
		}
	}

	rule2 := [][]int{}
	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, i+(basePow4*(j-1))+(basePow2*(k-1)))
			}
			rule2 = append(rule2, r)
		}
	}
	for _, r := range rule2 {
		cnfInt = append(cnfInt, r)
	}
	for _, r := range rule2 {
		c := combinations(r)
		for _, s := range c {
			cnfInt = append(cnfInt, s)
		}
	}

	rule3 := [][]int{}
	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, i+(basePow2*(j-1))+(basePow4*(k-1)))
			}
			rule3 = append(rule3, r)
		}
	}
	for _, r := range rule3 {
		cnfInt = append(cnfInt, r)
	}
	for _, r := range rule3 {
		c := combinations(r)
		for _, s := range c {
			cnfInt = append(cnfInt, s)
		}
	}

	rule4 := [][]int{}
	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			r := []int{}
			for k := 1; k <= basePow2; k++ {
				r = append(r, i+((j-1)%base)*base*basePow2+int((j-1)/base)*base*basePow4+((k-1)%base)*basePow2+int((k-1)/base)*(basePow2*basePow2))
			}
			rule4 = append(rule4, r)
		}
	}
	for _, r := range rule4 {
		cnfInt = append(cnfInt, r)
	}
	for _, r := range rule4 {
		c := combinations(r)
		for _, s := range c {
			cnfInt = append(cnfInt, s)
		}
	}

	for i := 1; i <= basePow2; i++ {
		for j := 1; j <= basePow2; j++ {
			if input[i-1][j-1] != 0 {
				s := []int{(i-1)*basePow4 + (j-1)*basePow2 + input[i-1][j-1]}
				cnfInt = append(cnfInt, s)
			}
		}
	}

	fmt.Println("Generated CNF is")
	fmt.Println(cnfInt)
	fmt.Printf("Generated CNF clause is %v\n", len(cnfInt))

	formula := cnf.NewFormulaFromInts(cnfInt)
	s := sat.New()
	s.AddFormula(formula)
	r := s.Solve()
	fmt.Printf("SAT: %v\n", r)
	fmt.Println("Assignments are")
	as := s.Assignments()
	keys := []int{}
	for k, a := range as {
		if a {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	for n, k := range keys {
		fmt.Println(n, k)
	}
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
