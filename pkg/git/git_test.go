package git

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"
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
		fmt.Printf("successfully checked out gitRef: %v\n", branchName)
	}

	cmd := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD")
	cmd.Dir = repoPath
	if out, err := cmd.Output(); err != nil {
		t.Fatalf("error executing git command, error: %v", err)
	} else if actualBranchName := strings.TrimSpace(string(out)); actualBranchName != branchName {
		t.Fatalf("expected branch to be: %v, got: %v", branchName, actualBranchName)
	} else {
		fmt.Printf("successfully created branch: %v, cleaning up\n", branchName)
		_ = os.RemoveAll(tmpDir)
	}
}

func TestDiscardAllUnstagedChanges(t *testing.T) {
	repo := "https://github.com/test/HelloWorld"

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	} else {
		fmt.Printf("created tmp dir: %v\n", tmpDir)
	}

	repoPath := path.Join(tmpDir, "repo")
	readmeBuffer := &bytes.Buffer{}

	r, err := CloneRepository(repoPath, repo, nil)
	if err != nil {
		t.Fatalf("error cloning repo: %v, error: %v", repo, err)
	}

	randomSalad := fmt.Sprintf("\nwith this salad: %v\n", time.Now())

	if err := ioutil.WriteFile(path.Join(repoPath, "salad"), []byte("greens\n"), os.ModePerm); err != nil {
		t.Fatalf("error adding salad to the repo: %v", err)
	} else if readmeBytes, err := ioutil.ReadFile(path.Join(repoPath, "README.md")); err != nil {
		t.Fatalf("error reading README.md from the repo: %v", err)
	} else if _, err := readmeBuffer.Write(readmeBytes); err != nil {
		t.Fatalf("error writing to buffer 1: %v", err)
	} else if _, err := readmeBuffer.Write([]byte(randomSalad)); err != nil {
		t.Fatalf("error writing to buffer 2: %v", err)
	} else if err := ioutil.WriteFile(path.Join(repoPath, "README.md"), readmeBuffer.Bytes(), os.ModePerm); err != nil {
		t.Fatalf("error adding salad to the repo's README.md: %v", err)
	}

	if err := DiscardAllUnstagedChanges(r); err != nil {
		t.Fatalf("error discarding changes to the repo: %v", err)
	} else if _, err := os.Stat(path.Join(repoPath, "salad")); !os.IsNotExist(err) {
		t.Fatal("expected salad to be gone from the repo, but it was still there")
	} else if readmeBytes, err := ioutil.ReadFile(path.Join(repoPath, "README.md")); err != nil {
		t.Fatalf("error reading README.md from the repo: %v", err)
	} else if bytes.HasSuffix(readmeBytes, []byte(randomSalad)) {
		t.Fatalf("expected salad to be gone from the repo's README.md, but it was still there")
	} else {
		fmt.Print("successfully discarded all unstaged changes\n")
		_ = os.RemoveAll(tmpDir)
	}
}

func TestGetRemoteReferences(t *testing.T) {
	repo := "https://github.com/qlik-oss/qliksense-k8s"

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	} else {
		//fmt.Printf("created tmp dir: %v\n", tmpDir)
		defer os.RemoveAll(tmpDir)
	}

	repoPath := path.Join(tmpDir, "repo")

	r, err := CloneRepository(repoPath, repo, nil)
	if err != nil {
		t.Fatalf("error cloning repo: %v, error: %v", repo, err)
	}

	remoteReferencesList, err := GetRemoteReferences(r, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(remoteReferencesList) != 1 {
		t.Fatal("expected remoteReferencesList to have size 1")
	}

	if len(remoteReferencesList[0].branches) < 1 {
		t.Fatal("expected there to be at least 1 branch")
	} else {
		t.Logf("branches: %v\n", remoteReferencesList[0].branches)
		foundMaster := false
		for _, branch := range remoteReferencesList[0].branches {
			if branch == "master" {
				foundMaster = true
			}
		}
		if !foundMaster {
			t.Fatal("expected the list of branches to contain master")
		}
	}

	if len(remoteReferencesList[0].tags) < 1 {
		t.Fatal("expected there to be at least 1 branch")
	} else {
		t.Logf("tags: %v\n", remoteReferencesList[0].tags)
	}
}
