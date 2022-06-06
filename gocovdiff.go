package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type flags struct {
	d string
	c string
	m string
	f string
}

func parseFlags() flags {
	var f flags

	flag.StringVar(&f.d, "diff", "", "Git diff file for changes (optional)")
	flag.StringVar(&f.c, "cov", "coverage.txt", "Coverage file")
	flag.StringVar(&f.m, "mod", "", "Module name (optional)")
	flag.StringVar(&f.f, "gha-annotations", "", "File to store GitHub Actions annotations")

	flag.Parse()

	if f.c == "" {
		flag.Usage()
		os.Exit(0)
	}

	return f
}

type line struct {
	statements int
	covered    int
}

func main() {
	if err := run(parseFlags()); err != nil {
		log.Fatal(err)
	}
}

func run(f flags) (err error) {
	if f.m == "" {
		o, err := exec.Command("go", "list", "-m").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get module name: %w", err)
		}

		f.m = strings.TrimSpace(string(o))
	}

	diff, err := getDiff(f.d)
	if err != nil {
		return err
	}

	var ga githubAnnotator

	if f.f != "" {
		f, err := os.Create(f.f)
		if err != nil {
			return fmt.Errorf("failed to create GitHub annotations file: %w", err)
		}

		ga.w = f

		defer func() {
			if err := f.Close(); err != nil {
				log.Fatal("failed to close GitHub annotations file: ", err)
			}
		}()
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
	totStmt := 0
	covStmt := 0
	fileCoverage := map[string]stat{}

	err = parseProfiles(f.c, func(fn string, block profileBlock) {
		fn = strings.TrimPrefix(fn, f.m+"/")
		testedFiles[fn] = true
		fStat := fileCoverage[fn]

		lines, ok := modified[fn]
		if !ok {
			return
		}

		stmtCaptured := false
		for i := block.StartLine; i <= block.EndLine; i++ {
			l, ok := lines[i]
			if !ok {
				continue
			}

			l.covered = block.Count
			if !stmtCaptured {
				l.statements += block.NumStmt
				totStmt += block.NumStmt
				fStat.totStmt += block.NumStmt

				if l.covered > 0 {
					covStmt += block.NumStmt
					fStat.covStmt += block.NumStmt
				}

				stmtCaptured = true
			}

			lines[i] = l
		}

		fileCoverage[fn] = fStat
	})
	if err != nil {
		return fmt.Errorf("failed to parse profiles: %w", err)
	}

	files := make([]string, 0, len(modified))
	for fn := range modified {
		files = append(files, fn)
	}

	sort.Strings(files)

	var functions []stat

	for _, fn := range files {
		if !testedFiles[fn] {
			ga.printNotTested(fn)
		}

		lines := modified[fn]

		ll := make([]int, 0, len(lines))

		for i, l := range lines {
			if l.covered == 0 {
				ll = append(ll, i)
			}
		}

		if len(ll) > 0 {
			sort.Ints(ll)

			p := ll[0]
			start := 0
			stmt := 0

			for _, i := range ll {
				if start == 0 {
					start = p
				}

				if i-p > 1 {
					ga.printNotice(fn, start, p, stmt)

					start = 0
					stmt = 0
				}

				stmt += lines[i].statements
				p = i
			}

			if start == 0 {
				start = p
			}

			ga.printNotice(fn, start, p, stmt)
		}

		funcs, err := findFuncs(fn)
		if err != nil {
			return fmt.Errorf("failed to find functions: %w", err)
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
				functions = append(functions, stat{
					name:       fu.name,
					file:       fn,
					line:       fu.startLine,
					covPercent: float64(covStmt) / float64(totStmt) * 100,
				})
			}
		}
	}

	printReport(os.Stdout, covStmt, totStmt, functions, fileCoverage)

	return nil
}

type stat struct {
	name             string
	file             string
	line             int
	covPercent       float64
	covStmt, totStmt int
}
