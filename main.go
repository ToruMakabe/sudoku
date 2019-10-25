package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"gonum.org/v1/gonum/stat/combin"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
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

	base := rowSize
	basePow := int(math.Pow(float64(base), float64(2)))
	cnfInt := [][]int{}

	rule1 := [][]int{}
	for i := 1; i <= basePow; i++ {
		r := []int{}
		for j := 1; j <= base; j++ {
			r = append(r, j+(base*(i-1)))
		}
		rule1 = append(rule1, r)
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

	fmt.Println("Generated CNF is")
	fmt.Println(cnfInt)
	fmt.Printf("Generated CNF clause is %v\n", len(cnfInt))

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
