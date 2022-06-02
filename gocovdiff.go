package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/waigani/diffparser"
)

func main() {
	var (
		d string
		c string
		m string
		f string
	)

	flag.StringVar(&d, "diff", "", "Code git diff file")
	flag.StringVar(&c, "cov", "coverage.txt", "Coverage file")
	flag.StringVar(&m, "mod", "", "Module name")
	flag.StringVar(&f, "func", "", "File to store function coverage report")

	flag.Parse()

	if d == "" || c == "" || m == "" {
		flag.Usage()
		os.Exit(0)
	}

	df, err := ioutil.ReadFile(d)
	if err != nil {
		log.Fatal(err)
	}

	diff, err := diffparser.Parse(string(df))
	if err != nil {
		log.Fatal(err)
	}

	type line struct {
		statements int
		covered    int
	}

	modified := map[string]map[int]line{}

	for _, f := range diff.Files {
		if !strings.HasSuffix(f.NewName, ".go") || strings.HasSuffix(f.NewName, "_test.go") {
			continue
		}

		lines := map[int]line{}

		for _, h := range f.Hunks {
			for _, l := range h.NewRange.Lines {
				lines[l.Number] = line{covered: -1}
			}
		}

		modified[f.NewName] = lines
	}

	testedFiles := map[string]bool{}

	err = parseProfiles(c, func(fn string, block profileBlock) {
		fn = strings.TrimPrefix(fn, m+"/")
		testedFiles[fn] = true

		lines, ok := modified[fn]
		if !ok {
			return
		}

		for i := block.StartLine; i <= block.EndLine; i++ {
			l, ok := lines[i]
			if !ok {
				continue
			}

			l.covered = block.Count
			l.statements += block.NumStmt
			lines[i] = l
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

	var functions []funcInfo

	for _, fn := range files {
		if !testedFiles[fn] {
			printNotTested(fn)
		}

		lines := modified[fn]

		if f != "" {
			funcs, err := findFuncs(fn)
			if err != nil {
				log.Fatal(err)
			}

			for _, fu := range funcs {
				totStmt := 0
				covStmt := 0

				for i := fu.startLine; i <= fu.endLine; i++ {
					if l, ok := lines[i]; ok {
						totStmt += l.statements
						if l.covered > 0 {
							covStmt += l.statements
						}
					}
				}

				if totStmt > 0 {
					functions = append(functions, funcInfo{
						name:       fu.name,
						file:       fn,
						covPercent: float64(covStmt) / float64(totStmt) * 100,
					})
				}
			}

			sort.Slice(functions, func(i, j int) bool {
				return functions[i].covPercent > functions[j].covPercent
			})

			res := "| File | Function | Coverage |\n"
			res += "| ---- | -------- | -------- |\n"
			for _, fu := range functions {
				res += fmt.Sprintf("| %s | %s | %.2f%% |\n", fu.file, fu.name, fu.covPercent)
			}

			err = ioutil.WriteFile(f, []byte(res), 0600)
			if err != nil {
				log.Fatal(err)
			}
		}

		ll := make([]int, 0, len(lines))

		for i, l := range lines {
			if l.covered == 0 {
				ll = append(ll, i)
			}
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

type funcInfo struct {
	name       string
	file       string
	covPercent float64
}

func printNotTested(fn string) {
	fmt.Println(fn, "not tested")
	fmt.Printf("::notice file=%s::File is not covered by tests.\n", fn)
}

func printNotice(fn string, start, end int) {
	fmt.Println(fn, start, end)
	fmt.Printf("::notice file=%s,line=%d,endLine=%d::Not covered by tests.\n", fn, start, end)
}

// profileBlock represents a single block of profiling data.
type profileBlock struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	NumStmt, Count      int
}

var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

// parseProfiles parses profile data in the specified file and calls a
// function for each Profile for each source file described therein.
// See https://github.com/golang/go/blob/0104a31b8fbcbe52728a08867b26415d282c35d2/src/cmd/cover/profile.go.
func parseProfiles(fileName string, cb func(fn string, block profileBlock)) error {
	pf, err := os.Open(fileName)
	if err != nil {
		return err
	}

	defer func() {
		if err := pf.Close(); err != nil {
			log.Fatal(err)
		}
	}()

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

		pb := profileBlock{
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
