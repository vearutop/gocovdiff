package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

func forkPointFromLocal() (string, error) {
	o, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git symbolic-ref refs/remotes/origin/HEAD: %w", err)
	}

	baseBranch := strings.TrimSpace(strings.TrimPrefix(string(o), "refs/remotes/"))

	o, err = exec.Command("git", "merge-base", baseBranch, "--fork-point").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git merge-base %s --fork-point: %w", baseBranch, err)
	}

	return strings.TrimSpace(string(o)), nil
}

func forkPointFromGitHub(eventPath string) (string, error) {
	f, err := ioutil.ReadFile(eventPath)
	if err != nil {
		return "", err
	}
	// pull_request.base.sha
	type event struct {
		PullRequest struct {
			Base struct {
				SHA string `json:"sha"`
			} `json:"base"`
		} `json:"pull_request"`
	}

	var e event
	if err := json.Unmarshal(f, &e); err != nil {
		println(string(f))

		return "", err
	}

	return e.PullRequest.Base.SHA, nil
}
