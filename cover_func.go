package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func reportCoverFuncDiff(w io.Writer, base, cur []byte) error {
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
		cf.curPercent = "0"
		res[cf.filename+":"+cf.funcname] = cf
	}

	for _, cf := range curCov {
		base := res[cf.filename+":"+cf.funcname]
		base.funcname = cf.funcname
		base.filename = cf.filename
		base.curPercent = cf.percent

		if base.percent == "" {
			base.percent = "0"
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
	data = append(data, []string{"Total", "", cf.percent, cf.curPercent})

	for _, fn := range funcs {
		cf := res[fn]
		data = append(data, []string{cf.filename, cf.funcname, cf.percent, cf.curPercent})
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

// coverFuncLineRe represents a line in a `go tool cover -func` output.
//    sample/foo.go:5:	foo		44.4%
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
