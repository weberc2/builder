package git

import (
	"io"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func gitCloneBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	var repo, sha string
	if err := dag.Inputs.VisitKeys(
		core.KeySpec{Key: "repo", Value: core.ParseString(&repo)},
		core.KeySpec{Key: "sha", Value: core.ParseString(&sha)},
	); err != nil {
		return errors.Wrap(err, "Parsing git_clone inputs")
	}

	if _, err := cache.TempDir(
		func(tmpDir string) (string, core.ArtifactID, error) {
			r, err := git.PlainClone(
				tmpDir,
				false,
				&git.CloneOptions{URL: string(repo)},
			)
			if err != nil {
				return "", core.ArtifactID{}, errors.Wrap(err, "Cloning repo")
			}

			worktree, err := r.Worktree()
			if err != nil {
				return "", core.ArtifactID{}, errors.Wrap(
					err,
					"Getting worktree",
				)
			}

			if err := worktree.Checkout(&git.CheckoutOptions{
				Hash:  plumbing.NewHash(string(sha)),
				Force: true,
			}); err != nil {
				return "", core.ArtifactID{}, errors.Wrapf(
					err,
					"Checking out sha %s",
					sha,
				)
			}

			return "", dag.ID.ArtifactID(), nil
		},
	); err != nil {
		return errors.Wrap(err, "Cloning git repo")
	}

	return nil
}

var Clone = core.Plugin{Type: "git_clone", BuildScript: gitCloneBuildScript}
