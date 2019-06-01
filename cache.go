package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Cache interface {
	Write(id ArtifactID, f func(io.Writer) error) error
	Exists(id ArtifactID) error
}

type LocalCache struct {
	Directory string
}

func (lc LocalCache) Write(id ArtifactID, f func(w io.Writer) error) error {
	artifactPath := filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		id.FilePath,
		fmt.Sprint(id.Checksum),
	)

	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(artifactPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return f(file)
}

var ErrArtifactNotFound = errors.New("Artifact not found")

func (lc LocalCache) Exists(id ArtifactID) error {
	_, err := os.Stat(filepath.Join(
		lc.Directory,
		string(id.Package),
		string(id.Target),
		id.FilePath,
		fmt.Sprint(id.Checksum),
	))
	if os.IsNotExist(err) {
		return ErrArtifactNotFound
	}
	return err
}
