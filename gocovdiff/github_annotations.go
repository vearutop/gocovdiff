package gocovdiff

import (
	"fmt"
	"io"
	"log"
)

type githubAnnotator struct {
	w io.Writer
}

func (a githubAnnotator) printNotTested(fn string) {
	if a.w == nil {
		return
	}

	_, err := fmt.Fprintf(a.w, "File %s is not covered by tests\n"+
		"::notice file=%s::File is not covered by tests.\n", fn, fn)
	if err != nil {
		log.Fatal("failed to write annotation: ", err)
	}
}

func (a githubAnnotator) printNotice(fn string, start, end int, lines map[int]*profileBlock) {
	if a.w == nil {
		return
	}

	b := lines[start]

	for i := start; i <= end; i++ {
		pb := lines[i]
		if pb.EndLine != b.EndLine {
			b.NumStmt += pb.NumStmt
			b.EndLine = pb.EndLine
		}
	}

	var err error

	if b.StartLine != start || b.EndLine != end {
		_, err = fmt.Fprintf(a.w, "%s:%d,%d: %d statement(s) on lines %d:%d are not covered by tests\n"+
			"::notice file=%s,line=%d,endLine=%d::%d statement(s) on lines %d:%d are not covered by tests.\n",
			fn, start, end, b.NumStmt, b.StartLine, b.EndLine,
			fn, start, end, b.NumStmt, b.StartLine, b.EndLine)
	} else {
		_, err = fmt.Fprintf(a.w, "%s:%d,%d: %d statement(s) are not covered by tests\n"+
			"::notice file=%s,line=%d,endLine=%d::%d statement(s) are not covered by tests.\n",
			fn, start, end, b.NumStmt,
			fn, start, end, b.NumStmt)
	}

	if err != nil {
		log.Fatal("failed to write annotation: ", err)
	}
}
