package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

var ErrArtifactNotFound = errors.New("Artifact not found")

type Cache func(id ArtifactID) string

func (c Cache) Exists(id ArtifactID) error {
	if _, err := os.Stat(c(id)); err != nil {
		if os.IsNotExist(err) {
			return ErrArtifactNotFound
		}
		return err
	}
	return nil
}

func (c Cache) Path(id ArtifactID) string { return c(id) }

func (c Cache) Open(id ArtifactID) (*os.File, error) {
	return os.Open(c(id))
}

func (c Cache) Read(id ArtifactID, f func(r io.Reader) error) error {
	file, err := c.Open(id)
	if err != nil {
		return err
	}
	defer file.Close()

	return f(file)
}

func (c Cache) Write(id ArtifactID, f func(w io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(c.Path(id)), 0755); err != nil {
		return err
	}

	file, err := os.Create(c.Path(id))
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("ERROR closing file %v", err)
		}
	}()

	return f(file)
}

func withTempDir(f func(dir string) error) error {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	if err := f(dir); err != nil {
		return err
	}

	if err := os.Remove(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	return nil
}

// TempDir provisions a temporary directory for the callback `f`. This manages
// cleaning up the temporary directory when `f` finishes. If `f` succeeds, it
// will return a path relative to the temporary directory that should be moved
// into the cache (the first return argument) and an artifact ID that is the
// address in the cache where the relative path should be moved. If `f` fails,
// the temporary directory is cleaned up and nothing is moved into the cache.
func (c Cache) TempDir(
	f func(dir string) (string, ArtifactID, error),
) (ArtifactID, error) {
	var aid ArtifactID
	err := withTempDir(func(dir string) error {
		relpath, id, err := f(dir)
		if err != nil {
			if removeErr := os.Remove(dir); removeErr != nil {
				return errors.Wrapf(
					removeErr,
					"Removing temporary directory while handling err: %v",
					err,
				)
			}
			return err
		}
		aid = id
		cachePath := c.Path(id)
		cacheParentDir := filepath.Dir(cachePath)
		if err := os.MkdirAll(cacheParentDir, 0755); err != nil {
			return errors.Wrapf(
				err,
				"Creating parent directory '%s' in cache",
				cacheParentDir,
			)
		}

		if err := os.RemoveAll(cachePath); err != nil {
			return errors.Wrap(err, "Removing old cache directory")
		}

		if err := os.Rename(
			filepath.Join(dir, relpath),
			cachePath,
		); err != nil {
			return errors.Wrap(err, "Moving temporary directory into cache")
		}

		return nil
	})
	return aid, err
}

func LocalCache(directory string) Cache {
	return func(id ArtifactID) string {
		if id.Target == "" {
			return filepath.Join(
				directory,
				"packages",
				string(id.Package),
				"filegroups",
				fmt.Sprint(id.Checksum),
			)
		}
		if id.Package == "" {
			id.Package = "__ROOT__"
		}
		return filepath.Join(
			directory,
			"packages",
			string(id.Package),
			"targets",
			string(id.Target),
			fmt.Sprint(id.Checksum),
		)
	}
}
