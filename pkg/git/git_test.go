package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
)

func TestCloneAndCheckout(t *testing.T) {
	repo := "https://github.com/test/HelloWorld"
	gitRef := "asd"

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	} else {
		fmt.Printf("created tmp dir: %v\n", tmpDir)
	}

	if repo, err := CloneRepository(path.Join(tmpDir, "repo"), repo, nil); err != nil {
		t.Fatalf("error cloning repo: %v, error: %v", repo, err)
	} else if err := Checkout(repo, gitRef, "", nil); err != nil {
		t.Fatalf("error checking out gitRef: %v, error: %v", gitRef, err)
	} else {
		fmt.Printf("successfully checked out gitRef: %v, cleaning up\n", gitRef)
		_ = os.RemoveAll(tmpDir)
	}
}

func TestBranchOnCheckout(t *testing.T) {
	repo := "https://github.com/test/HelloWorld"
	branchName := "asd"

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	} else {
		fmt.Printf("created tmp dir: %v\n", tmpDir)
	}

	repoPath := path.Join(tmpDir, "repo")
	if repo, err := CloneRepository(repoPath, repo, nil); err != nil {
		t.Fatalf("error cloning repo: %v, error: %v", repo, err)
	} else if err := Checkout(repo, branchName, branchName, nil); err != nil {
		t.Fatalf("error checking out gitRef: %v, error: %v", branchName, err)
	} else {
		fmt.Printf("successfully checked out gitRef: %v, cleaning up\n", branchName)
	}

	cmd := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD")
	cmd.Dir = repoPath
	if out, err := cmd.Output(); err != nil {
		t.Fatalf("error executing git command, error: %v", err)
	} else if actualBranchName := strings.TrimSpace(string(out)); actualBranchName != branchName {
		t.Fatalf("expected branch to be: %v, got: %v", branchName, actualBranchName)
	}
}
