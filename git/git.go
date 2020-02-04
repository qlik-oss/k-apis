package git

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
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

func Checkout(r *git.Repository, ref string, toBranch string, auth transport.AuthMethod) error {
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

//func ConfigureWorkTree(r *git.Repository) (*git.Worktree, error) {
//	w, err := r.Worktree()
//	if err != nil {
//		return nil, err
//	}
//
//	err = w.Checkout(&git.CheckoutOptions{
//		Branch: "refs/heads/master",
//		Force:  true,
//	})
//	if err != nil {
//		return nil, err
//	}
//	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
//	if err != git.NoErrAlreadyUpToDate && err != nil {
//		return nil, err
//	}
//	return w, nil
//}
//
//func CreateBranch(w *git.Worktree) (plumbing.ReferenceName, error) {
//	reference := plumbing.NewBranchReferenceName(
//		fmt.Sprintf("pr-branch-%s", tokenGenerator()),
//	)
//	b := plumbing.ReferenceName(reference)
//
//	err := w.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: b})
//
//	if err != nil {
//		err := w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b})
//		if err != git.NoErrAlreadyUpToDate && err != nil {
//			return "", err
//		}
//	}
//	return b, nil
//}
//
//func AddCommit(cr *config.CRSpec, w *git.Worktree) error {
//	_, err := w.Add(".")
//	if err != nil {
//		return err
//	}
//
//	_, err = w.Commit("k-apis pr", &git.CommitOptions{
//		Author: &object.Signature{
//			Name: cr.Git.UserName,
//			When: time.Now(),
//		},
//	})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func Push(cr *config.CRSpec, r *git.Repository) error {
//	err := r.Push(&git.PushOptions{
//		Auth: &http.BasicAuth{
//			Username: cr.Git.UserName,
//			Password: cr.Git.Password,
//		},
//	})
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func tokenGenerator() string {
//	b := make([]byte, 4)
//	rand.Read(b)
//	return fmt.Sprintf("%x", b)
//}
