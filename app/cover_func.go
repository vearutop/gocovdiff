package app

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func reportUndercoveredFuncs(w io.Writer, max float64, cur []byte) error {
	curCov, err := parseCoverFunc(cur)
	if err != nil {
		return fmt.Errorf("failed to parse current func coverage: %w", err)
	}

	data := make([][]string, 0, len(curCov))

	for _, cf := range curCov {
		if cf.funcname == "(statements)" {
			continue
		}

		c, err := strconv.ParseFloat(cf.percent, 64)
		if err != nil {
			return fmt.Errorf("failed to parse percent %q: %w", cf.percent, err)
		}

		if c > max {
			continue
		}

		data = append(data, []string{cf.filename, cf.funcname, cf.percent + "%"})
	}

	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"File", "Function", "Coverage"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()

	return nil
}

func reportCoverFuncDiff(w io.Writer, module string, base, cur []byte) error {
	baseCov, err := parseCoverFunc(base)
	if err != nil {
		return fmt.Errorf("failed to parse base func coverage: %w", err)
	}

	curCov, err := parseCoverFunc(cur)
	if err != nil {
		return fmt.Errorf("failed to parse current func coverage: %w", err)
	}

	res := make(map[string]coverFunc, len(curCov))

	for _, cf := range baseCov {
		cf.curPercent = "-"
		res[cf.filename+":"+cf.funcname] = cf
	}

	for _, cf := range curCov {
		base := res[cf.filename+":"+cf.funcname]
		base.funcname = cf.funcname
		base.filename = cf.filename
		base.curPercent = cf.percent

		if base.percent == "" {
			base.percent = "-"
		}

		res[cf.filename+":"+cf.funcname] = base
	}

	total := res["total:(statements)"]
	delete(res, "total:(statements)")

	funcs := make([]string, 0, len(res))
	for k := range res {
		funcs = append(funcs, k)
	}

	sort.Strings(funcs)

	data := make([][]string, 0, len(funcs))

	cf := total
	data = append(data, []string{"Total", "", cf.percent + "%", fmtCov(cf.percent, cf.curPercent)})

	for _, fn := range funcs {
		cf := res[fn]

		if cf.percent == cf.curPercent {
			continue
		}

		if module != "" {
			cf.filename = strings.TrimPrefix(cf.filename, module+"/")
		}

		switch {
		case cf.curPercent == "-":
			data = append(data, []string{cf.filename, cf.funcname, cf.percent + "%", "no function"})
		case cf.percent == "-":
			data = append(data, []string{cf.filename, cf.funcname, "no function", cf.curPercent + "%"})
		default:
			data = append(data, []string{cf.filename, cf.funcname, cf.percent + "%", fmtCov(cf.percent, cf.curPercent)})
		}
	}

	if len(data) == 1 {
		if _, err := w.Write([]byte("No changes in coverage.\n")); err != nil {
			return fmt.Errorf("failed to write to coverage change report: %w", err)
		}

		return nil
	}

	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"File", "Function", "Base Coverage", "Current Coverage"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()

	return nil
}

func fmtCov(base, cur string) string {
	b, err := strconv.ParseFloat(base, 64)
	if err != nil {
		log.Fatal(err)
	}

	c, err := strconv.ParseFloat(cur, 64)
	if err != nil {
		log.Fatal(err)
	}

	d := fmt.Sprintf("%.1f%%", c-b)
	if c-b > 0 {
		d = "+" + d
	}

	return cur + "% (" + d + ")"
}

// coverFuncLineRe represents a line in a `go tool cover -func` output.
//
//	sample/foo.go:5:	foo		44.4%
var coverFuncLineRe = regexp.MustCompile(`^([^:]+):([0-9:]*)\s+([\w0-9\(\)]+)\s+([0-9\.]+)%$`)

type coverFunc struct {
	filename   string
	funcname   string
	percent    string
	curPercent string
}

func parseCoverFunc(data []byte) ([]coverFunc, error) {
	lines := bytes.Split(data, []byte("\n"))
	res := make([]coverFunc, 0, len(lines))

	for _, line := range lines {
		l := strings.TrimSpace(string(line))
		if l == "" {
			continue
		}

		m := coverFuncLineRe.FindStringSubmatch(string(line))
		if m == nil {
			return nil, fmt.Errorf("line %q doesn't match expected format: %v", string(line), coverFuncLineRe)
		}

		cf := coverFunc{
			filename: m[1],
			funcname: m[3],
			percent:  m[4],
		}

		res = append(res, cf)
	}

	return res, nil
}
