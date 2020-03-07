package buildutil

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

type BuildContext struct {
	DAG       core.DAG
	Cache     core.Cache
	Stdout    io.Writer
	Stderr    io.Writer
	Workspace string
	Output    string
}

func (ctx *BuildContext) Call(
	command string,
	dir string,
	env []string,
	args ...string,
) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = ctx.Stdout
	cmd.Stderr = ctx.Stderr
	cmd.Dir = dir
	cmd.Env = env
	return cmd.Run()
}

func Build(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	script func(*BuildContext) error,
) error {
	workspace, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "Creating temp dir")
	}
	defer os.Remove(workspace)

	data := make([]byte, 16)
	rand.Seed(time.Now().UnixNano())
	rand.Read(data)
	output := base64.RawURLEncoding.EncodeToString(data)

	if err := script(&BuildContext{
		DAG:       dag,
		Cache:     cache,
		Stdout:    stdout,
		Stderr:    stderr,
		Workspace: workspace,
		Output:    output,
	}); err != nil {
		return errors.Wrap(err, "Running build script")
	}

	if err := os.MkdirAll(
		filepath.Dir(cache.Path(dag.ID.ArtifactID())),
		0700,
	); err != nil {
		return errors.Wrap(err, "Creating artifact's parent dir in cache")
	}

	if err := os.Rename(
		filepath.Join(workspace, output),
		cache.Path(dag.ID.ArtifactID()),
	); err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf("Build script failed to create artifact")
		}
		return errors.Wrap(err, "Moving artifact into cache")
	}
	return nil
}
