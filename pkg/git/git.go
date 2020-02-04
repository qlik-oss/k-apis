package git

import (
	"context"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

func CloneRepository(path string, repoUrl string, auth transport.AuthMethod) (*git.Repository, error) {
	cloneOptions := &git.CloneOptions{URL: repoUrl}
	if auth != nil {
		cloneOptions.Auth = auth
	}
	return git.PlainClone(path, false, cloneOptions)
}

func OpenRepository(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}

func Checkout(r *git.Repository, ref string, toBranch plumbing.ReferenceName, auth transport.AuthMethod) error {
	if hash, err := resolveRevision(r, ref, auth); err != nil {
		return err
	} else if workTree, err := r.Worktree(); err != nil {
		return err
	} else if err := workTree.Checkout(&git.CheckoutOptions{
		Hash:   *hash,
		Branch: plumbing.ReferenceName(toBranch),
		Create: toBranch != "",
	}); err != nil {
		return err
	}

	return nil
}

func AddCommit(r *git.Repository, author string) error {
	workTree, err := r.Worktree()
	if err != nil {
		return err
	}
	_, err = workTree.Add(".")
	if err != nil {
		return err
	}

	_, err = workTree.Commit("k-apis pr", &git.CommitOptions{
		Author: &object.Signature{
			Name: author,
			When: time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func Push(r *git.Repository, auth transport.AuthMethod) error {
	err := r.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}
	return nil
}

func resolveRevision(r *git.Repository, ref string, auth transport.AuthMethod) (*plumbing.Hash, error) {
	hash, err := r.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		hash, err = resolveRemoteTagOrBranch(r, ref, auth)
		if err != nil {
			return nil, err
		}
	}
	return hash, nil
}

func resolveRemoteTagOrBranch(r *git.Repository, findRef string, auth transport.AuthMethod) (*plumbing.Hash, error) {
	listOptions := &git.ListOptions{}
	if auth != nil {
		listOptions.Auth = auth
	}
	if remotes, err := r.Remotes(); err != nil {
		return nil, err
	} else {
		for _, remote := range remotes {
			if refs, err := remote.List(listOptions); err != nil {
				return nil, err
			} else {
				for _, ref := range refs {
					if (ref.Name().IsTag() || ref.Name().IsBranch()) &&
						(ref.Name().String() == findRef || ref.Name().Short() == findRef) {
						hash := ref.Hash()
						return &hash, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("ref is not a remote tag/branch: %v", findRef)
	}
}

func CreatePR(path string, token string, username string, branch string) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	newPR := &github.NewPullRequest{
		Title:               github.String("k-apis PR"),
		Head:                github.String(branch),
		Base:                github.String("master"),
		Body:                github.String("auto generated pr"),
		MaintainerCanModify: github.Bool(true),
	}

	_, _, err := client.PullRequests.Create(context.Background(), username, filepath.Base(path), newPR)
	if err != nil {
		return err
	}
	return nil
}

func TokenGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
