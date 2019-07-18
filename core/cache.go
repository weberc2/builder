package core

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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
