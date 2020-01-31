package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/go-github/github"
	"github.com/qlik-oss/k-apis/config"
	"golang.org/x/oauth2"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func CreatePR(cr *config.CRConfig, branch plumbing.ReferenceName) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cr.Git.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	newPR := &github.NewPullRequest{
		Title:               github.String("k-apis PR"),
		Head:                github.String(branch.String()),
		Base:                github.String("master"),
		Body:                github.String("auto generated pr"),
		MaintainerCanModify: github.Bool(true),
	}

	_, _, err := client.PullRequests.Create(context.Background(), cr.Git.UserName, filepath.Base(cr.ManifestsRoot), newPR)
	if err != nil {
		return err
	}
	return nil
}

func CloneRepository(cr *config.CRConfig) (*git.Repository, error) {
	r, err := git.PlainClone(cr.ManifestsRoot, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: cr.Git.UserName,
			Password: cr.Git.Password,
		},
		URL: cr.Git.Repository,
	})
	if err != nil {
		return nil, err
	}
	return r, nil

}

func OpenRepository(cr *config.CRConfig) (*git.Repository, error) {
	r, err := git.PlainOpen(cr.ManifestsRoot)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func ConfigureWorkTree(r *git.Repository) (*git.Worktree, error) {
	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/master",
		Force:  true,
	})
	if err != nil {
		return nil, err
	}
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != git.NoErrAlreadyUpToDate && err != nil {
		return nil, err
	}
	return w, nil
}

func CreateBranch(w *git.Worktree) (plumbing.ReferenceName, error) {
	reference := plumbing.NewBranchReferenceName(
		fmt.Sprintf("pr-branch-%s", tokenGenerator()),
	)
	b := plumbing.ReferenceName(reference)

	err := w.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: b})

	if err != nil {
		err := w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b})
		if err != git.NoErrAlreadyUpToDate && err != nil {
			return "", err
		}
	}
	return b, nil
}

func AddCommit(cr *config.CRConfig, w *git.Worktree) error {
	_, err := w.Add(".")
	if err != nil {
		return err
	}

	_, err = w.Commit("k-apis pr", &git.CommitOptions{
		Author: &object.Signature{
			Name: cr.Git.UserName,
			When: time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func Push(cr *config.CRConfig, r *git.Repository) error {
	err := r.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: cr.Git.UserName,
			Password: cr.Git.Password,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func tokenGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
