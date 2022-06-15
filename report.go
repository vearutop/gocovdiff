package main

import (
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/olekukonko/tablewriter"
)

func printReport(w io.Writer, covStmt, totStmt int, functions []stat, fileCoverage map[string]stat) {
	if totStmt == 0 {
		_, err := w.Write([]byte("No changes in testable statements.\n"))
		if err != nil {
			log.Fatal("failed to write report: ", err)
		}

		return
	}

	sort.Slice(functions, func(i, j int) bool {
		fi := functions[i]
		fj := functions[j]

		if fi.file != fj.file {
			return fi.file < fj.file
		}

		if fi.covPercent != fj.covPercent {
			return fi.covPercent > fj.covPercent
		}

		return fi.line < fj.line
	})

	data := make([][]string, 0, len(functions))
	data = append(data, []string{"Total", "", fmt.Sprintf("%.1f%%", float64(covStmt)/float64(totStmt)*100)})

	prevFile := ""
	for _, fu := range functions {
		if fu.file != prevFile {
			fc := fileCoverage[fu.file]

			data = append(data, []string{fu.file, "", fmt.Sprintf("%.1f%%", float64(fc.covStmt*100)/float64(fc.totStmt))})
		}

		data = append(data, []string{fmt.Sprintf("%s:%d", fu.file, fu.line), fu.name, fmt.Sprintf("%.1f%%", fu.covPercent)})
		prevFile = fu.file
	}

	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"File", "Function", "Coverage"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
}
