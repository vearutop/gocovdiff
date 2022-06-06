package main

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

	_, err := fmt.Fprintf(a.w, "::notice file=%s::File is not covered by tests.\n", fn)
	if err != nil {
		log.Fatal("failed to write annotation: ", err)
	}
}

func (a githubAnnotator) printNotice(fn string, start, end, stmt int) {
	if a.w == nil {
		return
	}

	_, err := fmt.Fprintf(a.w, "::notice file=%s,line=%d,endLine=%d::%d statement(s) not covered by tests.\n", fn, start, end, stmt)
	if err != nil {
		log.Fatal("failed to write annotation: ", err)
	}
}
