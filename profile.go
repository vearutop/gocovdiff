package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

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
