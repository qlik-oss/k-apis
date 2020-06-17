package git

import (
	"crypto/rand"
	"fmt"
	"sort"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
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
	hash, err := resolveRevision(r, ref, auth)
	if err != nil {
		return err
	}
	workTree, err := r.Worktree()
	if err != nil {
		return err
	}

	checkoutOptions := &git.CheckoutOptions{
		Hash:  *hash,
		Force: true,
	}
	if toBranch != "" {
		checkoutOptions.Create = true
		checkoutOptions.Branch = plumbing.NewBranchReferenceName(toBranch)
	}

	if err := workTree.Checkout(checkoutOptions); err != nil {
		return err
	}

	//it seems that if the git history included a directory case change
	//(delete and re-add with different letter case), we need this second checkout:
	if refHead, err := r.Head(); err != nil {
		return err
	} else if branchName := refHead.Name(); branchName != "" {
		return workTree.Checkout(&git.CheckoutOptions{Branch: branchName, Force: true})
	} else {
		return workTree.Checkout(&git.CheckoutOptions{Hash: refHead.Hash(), Force: true})
	}
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

// func CreatePR(path string, token string, username string, branch string) error {
// 	ctx := context.Background()
// 	ts := oauth2.StaticTokenSource(
// 		&oauth2.Token{AccessToken: token},
// 	)
// 	tc := oauth2.NewClient(ctx, ts)
// 	client := github.NewClient(tc)

// 	newPR := &github.NewPullRequest{
// 		Title:               github.String("k-apis PR"),
// 		Head:                github.String(branch),
// 		Base:                github.String("master"),
// 		Body:                github.String("auto generated pr"),
// 		MaintainerCanModify: github.Bool(true),
// 	}

// 	_, _, err := client.PullRequests.Create(context.Background(), username, filepath.Base(path), newPR)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func TokenGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func DiscardAllUnstagedChanges(r *git.Repository) error {
	if workTree, err := r.Worktree(); err != nil {
		return err
	} else if refHead, err := r.Head(); err != nil {
		return err
	} else if err := workTree.Clean(&git.CleanOptions{Dir: true}); err != nil {
		return err
	} else if branchName := refHead.Name(); branchName != "" {
		return workTree.Checkout(&git.CheckoutOptions{Branch: branchName, Force: true})
	} else {
		return workTree.Checkout(&git.CheckoutOptions{Hash: refHead.Hash(), Force: true})
	}
}

type RemoteRefs struct {
	Name     string
	Branches []string
	Tags     []string
}

type RefSortOrder byte

const (
	RefSortOrderAscending RefSortOrder = iota
	RefSortOrderDescending
)

type RemoteRefConstraints struct {
	Include   bool
	Sort      bool
	SortOrder RefSortOrder
}

func sortStringSlice(data *[]string, sortOrder RefSortOrder) {
	if sortOrder == RefSortOrderAscending {
		sort.Sort(sort.StringSlice(*data))
	} else {
		sort.Sort(sort.Reverse(sort.StringSlice(*data)))
	}
}

func GetRemoteRefs(
	r *git.Repository,
	auth transport.AuthMethod,
	tagConstraints *RemoteRefConstraints,
	branchConstraints *RemoteRefConstraints) (remoteRefsList []*RemoteRefs, err error) {
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
				remoteReferences := &RemoteRefs{
					Name:     remote.Config().Name,
					Branches: []string{},
					Tags:     []string{},
				}
				remoteRefsList = append(remoteRefsList, remoteReferences)
				for _, ref := range refs {
					if tagConstraints.Include && ref.Name().IsTag() {
						remoteReferences.Tags = append(remoteReferences.Tags, ref.Name().Short())
					} else if branchConstraints.Include && ref.Name().IsBranch() {
						remoteReferences.Branches = append(remoteReferences.Branches, ref.Name().Short())
					}
				}
				if tagConstraints.Include && tagConstraints.Sort {
					sortStringSlice(&remoteReferences.Tags, tagConstraints.SortOrder)
				}
				if branchConstraints.Include && branchConstraints.Sort {
					sortStringSlice(&remoteReferences.Branches, branchConstraints.SortOrder)
				}
			}
		}
		return remoteRefsList, nil
	}
}
