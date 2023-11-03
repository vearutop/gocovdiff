package app

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bool64/dev/version"
	"github.com/waigani/diffparser"
)

type flags struct {
	diffFile       string
	parentCommit   string
	covFile        string
	module         string
	ghaAnnotations string
	exclude        string
	funcCov        string
	funcMaxCov     float64
	funcBaseCov    string
	targetDeltaCov float64
	deltaCovFile   string
	version        bool
}

func parseFlags() flags {
	var f flags

	flag.StringVar(&f.diffFile, "diff", "", "Git diff file for changes (optional)")
	flag.StringVar(&f.parentCommit, "parent", "", "Parent commit hash (optional)")
	flag.StringVar(&f.covFile, "cov", "coverage.txt", "Coverage file")
	flag.StringVar(&f.module, "mod", "", "Module name to strip from file names (optional)")
	flag.StringVar(&f.ghaAnnotations, "gha-annotations", "", "File to store GitHub Actions annotations")
	flag.StringVar(&f.exclude, "exclude", "", "Exclude directories by prefix and files by name pattern, comma separated (optional)")

	flag.StringVar(&f.funcCov, "func-cov", "", "Current func coverage from 'go tool cover -func', requires -func-base-cov or -func-max-cov (optional)")
	flag.StringVar(&f.funcBaseCov, "func-base-cov", "", "Base func coverage from 'go tool cover -func', requires -func-cov (optional)")
	flag.Float64Var(&f.funcMaxCov, "func-max-cov", 0, "Max func coverage from 'go tool cover -func' to keep in report of undercovered functions, requires -func-cov (optional)")

	flag.Float64Var(&f.targetDeltaCov, "target-delta-cov", 80, "Target coverage of changed lines, to be used together with -delta-cov-file")
	flag.StringVar(&f.deltaCovFile, "delta-cov-file", "", "File to store delta coverage message")

	flag.BoolVar(&f.version, "version", false, "Show version and exit")

	flag.Parse()

	if f.version {
		fmt.Println(version.Module("github.com/vearutop/gocovdiff").Version)
		os.Exit(0)
	}

	if f.covFile == "" {
		flag.Usage()
		os.Exit(0)
	}

	if f.funcMaxCov == 0 && (f.funcCov != "" || f.funcBaseCov != "") {
		if f.funcCov == "" || f.funcBaseCov == "" {
			flag.Usage()
			os.Exit(1)
		}
	}

	if f.funcMaxCov > 0 && f.funcCov == "" {
		flag.Usage()
		os.Exit(1)
	}

	return f
}

// Main runs application.
func Main() {
	if err := run(parseFlags(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

//nolint:maintidx
func run(f flags, report io.Writer) (err error) {
	if f.funcMaxCov > 0 && f.funcCov != "" {
		cur, err := os.ReadFile(f.funcCov)
		if err != nil {
			return fmt.Errorf("failed to read current coverage file: %w", err)
		}

		return reportUndercoveredFuncs(report, f.funcMaxCov, cur)
	}

	if f.funcCov != "" && f.funcBaseCov != "" {
		base, err := os.ReadFile(f.funcBaseCov)
		if err != nil {
			return fmt.Errorf("failed to read base coverage file: %w", err)
		}

		cur, err := os.ReadFile(f.funcCov)
		if err != nil {
			return fmt.Errorf("failed to read current coverage file: %w", err)
		}

		return reportCoverFuncDiff(report, f.module, base, cur)
	}

	if f.module == "" {
		o, err := exec.Command("go", "list", "-m").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get module name: %w", err)
		}

		f.module = strings.TrimSpace(string(o))
	}

	diff, err := getDiff(f.diffFile, f.parentCommit)
	if err != nil {
		return err
	}

	var ga githubAnnotator

	if f.ghaAnnotations != "" {
		f, err := os.Create(f.ghaAnnotations)
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

	modified := map[string]map[int]*profileBlock{}
	exclude := []string(nil)

	if f.exclude != "" {
		exclude = strings.Split(f.exclude, ",")
	}

fileLoop:
	for _, f := range diff.Files {
		if !strings.HasSuffix(f.NewName, ".go") || strings.HasSuffix(f.NewName, "_test.go") {
			continue
		}

		for _, e := range exclude {
			if strings.HasPrefix(f.NewName, e) {
				continue fileLoop
			}

			if ok, err := filepath.Match(e, filepath.Base(f.NewName)); ok && err == nil {
				continue fileLoop
			}
		}

		lines := map[int]*profileBlock{}
		fmt.Printf("DIFF FILE: %#v\n", f)

		for _, h := range f.Hunks {
			for _, l := range h.NewRange.Lines {
				if l.Mode == diffparser.UNCHANGED {
					continue
				}

				lines[l.Number] = &profileBlock{Count: -1}
			}
		}

		modified[f.NewName] = lines
	}

	testedFiles := map[string]bool{}
	totStmt := 0
	covStmt := 0
	fileCoverage := map[string]stat{}

	err = parseProfiles(f.covFile, func(fn string, block profileBlock) {
		fn = strings.TrimPrefix(fn, f.module+"/")
		testedFiles[fn] = true
		fStat := fileCoverage[fn]

		lines, ok := modified[fn]
		if !ok {
			return
		}

		totCounted := false
		for i := block.StartLine; i <= block.EndLine; i++ {
			l, ok := lines[i]
			if !ok {
				continue
			}

			if !totCounted {
				totStmt += block.NumStmt
				fStat.totStmt += block.NumStmt

				if block.Count > 0 {
					covStmt += block.NumStmt
					fStat.covStmt += block.NumStmt
				}

				totCounted = true
			}

			if l.Count == -1 {
				lines[i] = &block

				continue
			}

			// Do not merge blocks that has coverage.
			if block.Count > 0 {
				continue
			}

			l.NumStmt += block.NumStmt

			if l.StartLine == block.StartLine && l.StartCol > block.StartCol {
				l.StartCol = block.StartCol
			}

			if l.StartLine > block.StartLine {
				l.StartLine = block.StartLine
				l.StartCol = block.StartCol
			}

			if l.EndLine == block.EndLine && l.EndCol < block.EndCol {
				l.EndCol = block.EndCol
			}

			if l.EndLine < block.EndLine {
				l.EndLine = block.EndLine
				l.EndCol = block.EndCol
			}
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

	var (
		functions     []stat
		untestedFiles []string
	)

	for _, fn := range files {
		if !testedFiles[fn] {
			ga.printNotTested(fn)
			untestedFiles = append(untestedFiles, fn)
		}

		lines := modified[fn]

		ll := make([]int, 0, len(lines))

		for i, l := range lines {
			if l.Count == 0 {
				ll = append(ll, i)
			}
		}

		if len(ll) > 0 {
			sort.Ints(ll)

			p := ll[0]
			start := 0

			for _, i := range ll {
				if start == 0 {
					start = p
				}

				if i-p > 1 {
					ga.printNotice(fn, start, p, lines)

					start = 0
				}

				p = i
			}

			if start == 0 {
				start = p
			}

			ga.printNotice(fn, start, p, lines)
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
					totStmt += l.NumStmt

					if l.Count > 0 {
						covStmt += l.NumStmt
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

	printReport(report, covStmt, totStmt, functions, fileCoverage, untestedFiles)

	if f.deltaCovFile == "" {
		return nil
	}

	df, err := os.Create(f.deltaCovFile)
	if err != nil {
		return fmt.Errorf("failed to create delta coverage file: %w", err)
	}

	defer func() {
		if err := df.Close(); err != nil {
			log.Fatal("failed to close delta coverage file: ", err)
		}
	}()

	res := ""

	if totStmt > 0 {
		deltaCov := float64(covStmt) / float64(totStmt) * 100
		res = fmt.Sprintf("changed lines: (statements) %.1f%%", deltaCov)

		if deltaCov < f.targetDeltaCov {
			res += fmt.Sprintf(", coverage is less than %.1f%%, consider testing the changes more thoroughly", f.targetDeltaCov)
		}
	}

	if _, err = df.WriteString(res); err != nil {
		return fmt.Errorf("failed to write to delta coverage file: %w", err)
	}

	return nil
}

type stat struct {
	name             string
	file             string
	line             int
	covPercent       float64
	covStmt, totStmt int
}
