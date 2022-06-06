package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/waigani/diffparser"
)

func gitDiff() ([]byte, error) {
	var (
		forkPoint string
		err       error
	)

	if eventPath := os.Getenv("GITHUB_EVENT_PATH"); eventPath != "" {
		forkPoint, err = forkPointFromGitHub(eventPath)
	} else {
		forkPoint, err = forkPointFromLocal()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to file fork point: %w", err)
	}

	o, err := exec.Command("git", "diff", forkPoint, "--no-color").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff %s --no-color: %w\n%s", forkPoint, err, string(o))
	}

	return o, nil
}

func getDiff(diffFile string) (*diffparser.Diff, error) {
	var d []byte

	if diffFile == "" {
		o, err := gitDiff()
		if err != nil {
			return nil, err
		}

		d = o
	} else {
		df, err := ioutil.ReadFile(diffFile)
		if err != nil {
			log.Fatal(err)
		}
		d = df
	}

	diff, err := diffparser.Parse(string(d))
	if err != nil {
		return nil, fmt.Errorf("failed to parse git diff: %w", err)
	}

	return diff, nil
}
