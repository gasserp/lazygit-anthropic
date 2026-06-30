// Package git provides thin wrappers around the system git binary.
//
// Each helper shells out to the real `git` executable in the current working
// directory and returns its stdout. On a non-zero exit, the returned error
// includes git's stderr so callers can surface a useful message.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// run executes `git <args...>` and returns trimmed stdout. On failure it wraps
// the error together with git's stderr output.
func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

// StagedDiff returns the diff of staged changes (`git diff --cached`).
func StagedDiff() (string, error) {
	return run("diff", "--cached")
}

// StagedNameStatus returns the name-status summary of staged changes
// (`git diff --cached --name-status`).
func StagedNameStatus() (string, error) {
	return run("diff", "--cached", "--name-status")
}

// CurrentBranch returns the current branch name (`git rev-parse --abbrev-ref HEAD`).
func CurrentBranch() (string, error) {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// OriginHead resolves the default branch of origin via
// `git symbolic-ref --quiet refs/remotes/origin/HEAD`, returning the short
// branch name (e.g. "main"). Returns an empty string with no error when the
// ref is not set.
func OriginHead() (string, error) {
	out, err := run("symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err != nil {
		// Not configured: not a fatal condition for branch resolution.
		return "", nil
	}
	ref := strings.TrimSpace(out)
	return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
}

// RefExists reports whether the given ref resolves
// (`git rev-parse --verify --quiet <ref>`).
func RefExists(ref string) bool {
	_, err := run("rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

// Log returns the commit subjects and bodies in the range base..HEAD.
func Log(base string) (string, error) {
	return run("log", "--pretty=format:%s%n%n%b%n", base+"..HEAD")
}

// HasCommits reports whether there are any commits in base..HEAD.
func HasCommits(base string) (bool, error) {
	out, err := run("rev-list", "--count", base+"..HEAD")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "0", nil
}

// RangeDiff returns the three-dot diff between base and HEAD
// (`git diff base...HEAD`).
func RangeDiff(base string) (string, error) {
	return run("diff", base+"...HEAD")
}
