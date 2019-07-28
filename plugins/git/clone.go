package git

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

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
	repo, err := dag.Inputs.GetString("repo")
	if err != nil {
		return err
	}

	sha, err := dag.Inputs.GetString("sha")
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temp dir for git clone")
	}
	defer os.RemoveAll(tmpDir)

	r, err := git.PlainClone(
		tmpDir,
		false,
		&git.CloneOptions{URL: string(repo)},
	)
	if err != nil {
		return errors.Wrap(err, "Cloning repo")
	}

	worktree, err := r.Worktree()
	if err != nil {
		return errors.Wrap(err, "Getting worktree")
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Hash:  plumbing.NewHash(string(sha)),
		Force: true,
	}); err != nil {
		return errors.Wrapf(err, "Checking out sha %s", sha)
	}

	finalCachePath := cache.Path(dag.ID.ArtifactID())
	if err := os.MkdirAll(filepath.Dir(finalCachePath), 0755); err != nil {
		return errors.Wrap(err, "Creating parent directory in cache")
	}

	if err := os.Rename(tmpDir, finalCachePath); err != nil {
		return errors.Wrap(err, "Moving tmp dir to final cache location")
	}

	return nil
}

var Clone = core.Plugin{
	Type: "git_clone",
	Factory: func(core.FrozenObject) (core.BuildScript, error) {
		return gitCloneBuildScript, nil
	},
}
