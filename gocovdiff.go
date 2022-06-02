package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/waigani/diffparser"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	var (
		d string
		c string
		m string
	)

	flag.StringVar(&d, "diff", "", "Git diff file")
	flag.StringVar(&c, "cov", "coverage.txt", "Coverage file")
	flag.StringVar(&m, "mod", "", "Module name")

	flag.Parse()

	df, err := ioutil.ReadFile(d)
	if err != nil {
		log.Fatal(err)
	}

	diff, err := diffparser.Parse(string(df))
	if err != nil {
		log.Fatal(err)
	}

	type rng struct {
		start, end int
		covered    []int
	}

	modified := map[string]map[int]bool{}

	for _, f := range diff.Files {
		if !strings.HasSuffix(f.NewName, ".go") || strings.HasSuffix(f.NewName, "_test.go") {
			continue
		}

		lines := map[int]bool{}

		for _, h := range f.Hunks {
			for _, l := range h.NewRange.Lines {
				lines[l.Number] = true
				//println(f.NewName, l.Number, l.Content)
			}
		}

		modified[f.NewName] = lines
	}

	err = ParseProfiles(c, func(fn string, block ProfileBlock) {
		fn = strings.TrimPrefix(fn, m+"/")

		if block.Count == 0 {
			return
		}

		lines, ok := modified[fn]
		if !ok {
			return
		}

		//println("cov", fn, block.StartLine, block.EndLine)

		for i := block.StartLine; i <= block.EndLine; i++ {
			delete(lines, i)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	files := make([]string, 0, len(modified))
	for fn := range modified {
		files = append(files, fn)
	}

	sort.Strings(files)

	for _, fn := range files {
		lines := modified[fn]

		ll := make([]int, 0, len(lines))

		for i := range lines {
			ll = append(ll, i)
		}

		sort.Ints(ll)

		p := ll[0]
		start := 0
		for _, i := range ll {
			if start == 0 {
				start = p
			}

			if i-p > 1 {
				printNotice(fn, start, p)

				start = 0
			}

			p = i
		}

		if start == 0 {
			start = p
		}

		printNotice(fn, start, p)
	}
}

func printNotice(fn string, start, end int) {
	fmt.Println(fn, start, end)
	fmt.Printf("::notice file=%s,line=%d,endLine=%d::Not covered by tests.\n", fn, start, end)
}

// ProfileBlock represents a single block of profiling data.
type ProfileBlock struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	NumStmt, Count      int
}

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

// ParseProfiles parses profile data in the specified file and returns a
// Profile for each source file described therein.
func ParseProfiles(fileName string, cb func(fn string, block ProfileBlock)) error {
	pf, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer pf.Close()

	buf := bufio.NewReader(pf)
	// First line is "mode: foo", where foo is "set", "count", or "atomic".
	// Rest of file is in the format
	//	encoding/base64/base64.go:34.44,37.40 3 1
	// where the fields are: name.go:line.column,line.column numberOfStatements count
	s := bufio.NewScanner(buf)
	mode := ""
	for s.Scan() {
		line := s.Text()
		if mode == "" {
			const p = "mode: "
			if !strings.HasPrefix(line, p) || line == p {
				return fmt.Errorf("bad mode line: %v", line)
			}
			mode = line[len(p):]
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			return fmt.Errorf("line %q doesn't match expected format: %v", m, lineRe)
		}
		fn := m[1]

		pb := ProfileBlock{
			StartLine: toInt(m[2]),
			StartCol:  toInt(m[3]),
			EndLine:   toInt(m[4]),
			EndCol:    toInt(m[5]),
			NumStmt:   toInt(m[6]),
			Count:     toInt(m[7]),
		}

		cb(fn, pb)
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func toInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}
