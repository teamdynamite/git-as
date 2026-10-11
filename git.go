package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Exit codes that git config uses for expected "not found" cases.
const (
	exitKeyNotFound   = 1 // --get, --get-regexp: no matching key
	exitUnsetNotFound = 5 // --unset-all: no matching key
)

// gitError is a git command that exited with a non-zero status.
type gitError struct {
	args   []string
	code   int
	stderr string
}

func (e *gitError) Error() string {
	msg := "git " + strings.Join(e.args, " ") + " failed"
	if e.stderr != "" {
		msg += ": " + e.stderr
	}
	return msg
}

// git runs a git command and returns its stdout.
func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", &gitError{args: args, code: exitErr.ExitCode(), stderr: strings.TrimSpace(stderr.String())}
		}
		return "", fmt.Errorf("cannot run git: %w", err)
	}
	return stdout.String(), nil
}

// exitCode returns git's exit code from err, or -1 if err is not a gitError.
func exitCode(err error) int {
	var gerr *gitError
	if errors.As(err, &gerr) {
		return gerr.code
	}
	return -1
}

// configArgs builds a git config command line. An empty scope means the
// effective value across all scopes.
func configArgs(scope string, args ...string) []string {
	out := []string{"config"}
	if scope != "" {
		out = append(out, "--"+scope)
	}
	return append(out, args...)
}

// configGet returns the value of key, or "" if it is not set.
func configGet(scope, key string) (string, error) {
	out, err := git(configArgs(scope, "--get", key)...)
	if exitCode(err) == exitKeyNotFound {
		return "", nil
	}
	return strings.TrimSuffix(out, "\n"), err
}

// configSetOrUnset sets key to value, or unsets it when value is empty.
// Unsetting a key that is not set is not an error.
func configSetOrUnset(scope, key, value string) error {
	if value != "" {
		_, err := git(configArgs(scope, "--", key, value)...)
		return err
	}
	_, err := git(configArgs(scope, "--unset-all", key)...)
	if exitCode(err) == exitUnsetNotFound {
		return nil
	}
	return err
}

// configRemoveSection deletes a whole config section, e.g. "profile.frank".
func configRemoveSection(scope, section string) error {
	_, err := git(configArgs(scope, "--remove-section", section)...)
	return err
}

// insideRepo reports whether the working directory is inside a git repository.
func insideRepo() bool {
	_, err := git("rev-parse", "--git-dir")
	return err == nil
}
