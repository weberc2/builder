package python

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/weberc2/builder/core"
)

func installWheelPaths(
	path string,
	wheelPaths []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	// Run the `pip` in the virtualenv directory (passed in via `path`). This
	// should be sufficient to run this in the virtualenv, but we're also
	// updating the PATH environment variable to include the `path` directory as
	// well.
	cmd := exec.Command(
		filepath.Join(path, "pip"),
		append(
			[]string{"install", "--no-deps"}, wheelPaths...,
		)...,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = prependPATH(os.Environ(), path)
	return errors.Wrapf(
		cmd.Run(),
		"Installing wheels [%s]",
		strings.Join(wheelPaths, ", "),
	)
}

func deduplicate(sources []string) []string {
	seen := map[string]struct{}{}
	var output []string
	for _, source := range sources {
		if _, found := seen[source]; !found {
			output = append(output, source)
			seen[source] = struct{}{}
		}
	}
	return output
}

func findTarget(targets []core.DAG, id core.ArtifactID) (core.DAG, error) {
	for _, target := range targets {
		if target.ID.ArtifactID() == id {
			return target, nil
		}
	}
	return core.DAG{}, errors.Wrapf(ErrUnknownTarget, "Target = %s", id)
}

func gatherTargets(
	allTargets []core.DAG,
	ids []core.ArtifactID,
) ([]core.DAG, error) {
	var output []core.DAG
	for _, id := range ids {
		dag, err := findTarget(allTargets, id)
		if err != nil {
			return nil, err
		}
		output = append(output, dag)
	}
	return output, nil
}

func gatherWheelPaths(cache core.Cache, targets []core.DAG) ([]string, error) {
	var output []string
	for _, target := range targets {
		wheelPaths, err := fetchWheelPaths(cache, target)
		if err != nil {
			return nil, err
		}
		output = append(output, wheelPaths...)
	}
	return output, nil
}

func virtualEnvBuildScript(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
) error {
	var dependencies []core.ArtifactID
	if err := dag.Inputs.VisitKey(
		"dependencies",
		core.AssertArrayOf(core.AssertArtifactID(
			func(dep core.ArtifactID) error {
				dependencies = append(dependencies, dep)
				return nil
			},
		)),
	); err != nil {
		return errors.Wrap(err, "Parsing py_virtualenv inputs")
	}
	return virtualEnvPrepare(dag, cache, stdout, stderr, dependencies)
}

func prependPATH(environCopy []string, value string) []string {
	for i, entry := range environCopy {
		if strings.HasPrefix(entry, "PATH=") {
			environCopy[i] = fmt.Sprintf(
				"PATH=%s:%s",
				value,
				entry[len("PATH="):],
			)
			return environCopy
		}
	}
	// If we never found the PATH env var in the list of env vars, then
	// create a new one and append it to the list
	return append(environCopy, "PATH="+value)
}

type command struct {
	Command string
	Args    []string
	Stdout  io.Writer
	Stderr  io.Writer
	Dir     string
	Env     []string
}

func prepend(s string, ss ...string) []string {
	return append([]string{s}, ss...)
}

// Run the `python` from the virtualenv directory. This should be sufficient to
// run this in the virtualenv, but we're also updating the PATH environment
// variable to include the `venvBinDir` as well.
func venvCmd(
	cache core.Cache,
	venv core.ArtifactID,
	command command,
) *exec.Cmd {
	venvBinDir := filepath.Join(cache.Path(venv), "bin")
	pythonExe := filepath.Join(venvBinDir, "python")
	cmd := exec.Command(
		pythonExe,
		prepend("-m", prepend(command.Command, command.Args...)...)...,
	)
	cmd.Stdout = command.Stdout
	cmd.Stderr = command.Stderr
	cmd.Dir = command.Dir
	cmd.Env = prependPATH(command.Env, venvBinDir)
	return cmd
}

func replaceInFile(filePath, old, new string) error {
	fi, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		filePath,
		bytes.ReplaceAll(data, []byte(old), []byte(new)),
		fi.Mode(),
	)
}

// Prepares the `old` venv dir to be moved to filepath `new`. Since venvs have
// files that contain their absolute paths, it's imperative to replace those
// references with references to the new absolute paths. As such, this script
// does that find and replace before the `old` dir is moved to the `new` path,
// and this operation depends on `old` and `new` being absolute paths (this
// function does not verify, however). Also, an error can leave the `old` venv
// in a partially-moved state (references in the old files might be updated to
// their new paths).
func mvvenv(old, new string) error {
	// This function assumes that all files that _might_ contain references to
	// the old absolute path are included in this list.
	files := []string{
		"bin/easy_install",
		"bin/easy_install-3.6",
		"bin/pip",
		"bin/activate.fish",
		"bin/activate",
		"bin/activate.csh",
	}

	for _, file := range files {
		if err := replaceInFile(
			filepath.Join(old, file),
			old,
			new,
		); err != nil {
			return errors.Wrapf(
				err,
				"Replacing '%s' with '%s' in file '%s'",
				old,
				new,
				filepath.Join(old, file),
			)
		}
	}

	return nil
}

func virtualEnvPrepare(
	dag core.DAG,
	cache core.Cache,
	stdout io.Writer,
	stderr io.Writer,
	dependencies []core.ArtifactID,
) error {
	if _, err := cache.TempDir(
		func(dir string) (string, core.ArtifactID, error) {
			venvDir := filepath.Join(dir, ".venv")
			cmd := exec.Command("python", "-m", "venv", venvDir)
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			cmd.Dir = dir
			if err := cmd.Run(); err != nil {
				return "", core.ArtifactID{}, errors.Wrap(
					err,
					"Creating virtualenv",
				)
			}
			targets, err := gatherTargets(dag.Dependencies, dependencies)
			if err != nil {
				return "", core.ArtifactID{}, err
			}
			wheelPaths, err := gatherWheelPaths(cache, targets)
			if err != nil {
				return "", core.ArtifactID{}, err
			}
			wheelPaths = deduplicate(wheelPaths)

			if err := installWheelPaths(
				// Make sure `python` is the venv's python and not the system
				// python.
				filepath.Join(venvDir, "bin"),
				wheelPaths,
				stdout,
				stderr,
			); err != nil {
				return "", core.ArtifactID{}, errors.Wrapf(
					err,
					"Installing wheels [%s] into directory %s",
					strings.Join(wheelPaths, ", "),
					dir,
				)
			}

			// Since everything has succeeded, we can move the venv directory
			// into the cache.
			outputPath := cache.Path(dag.ID.ArtifactID())
			if err := mvvenv(venvDir, outputPath); err != nil {
				return "", core.ArtifactID{}, errors.Wrapf(
					err,
					"Moving venv dir from %s to %s",
					venvDir,
					outputPath,
				)
			}

			return ".venv", dag.ID.ArtifactID(), nil
		},
	); err != nil {
		return errors.Wrap(err, "Preparing virtualenv")
	}
	return nil
}

var VirtualEnv = core.Plugin{
	Type:        BuilderTypeVirtualEnv,
	BuildScript: virtualEnvBuildScript,
}
