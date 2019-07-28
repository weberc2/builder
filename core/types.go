package core

import (
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
)

type TargetID struct {
	Package PackageName
	Target  TargetName
}

type InvalidTargetIDErr string

func (err InvalidTargetIDErr) Error() string {
	return fmt.Sprintf("Invalid target ID: %s", string(err))
}

type PackageNameNotInWorkspaceErr struct {
	Workspace        string
	WorkingDirectory string
	PackageName      string
}

func (err PackageNameNotInWorkspaceErr) Error() string {
	return fmt.Sprintf(
		"Package name %s (relative to working directory %s) not in "+
			"workspace %s",
		err.PackageName,
		err.WorkingDirectory,
		err.Workspace,
	)
}

func ParseTargetID(workspace, cwd, s string) (TargetID, error) {
	i := strings.Index(string(s), ":")
	if i < 0 { // must have a colon
		return TargetID{}, InvalidTargetIDErr(s)
	}

	packageName := s[:i]

	// If it's a relative package path, join it to the working directory and
	// make it relative to the workspace
	if !strings.HasPrefix(packageName, "//") {
		result, err := filepath.Rel(workspace, filepath.Join(cwd, packageName))
		if err != nil {
			return TargetID{}, err
		}

		// In cases where the package name == cwd == workspace, result will be
		// '.' such that `//.` is considered different than `//` even though
		// these are clearly pointing to the same package. We need to normalize
		// this.
		if result == "." {
			result = ""
		}
		packageName = result
	} else {
		if strings.HasPrefix(packageName, "//./") {
			return TargetID{}, InvalidTargetIDErr(s)
		}
		packageName = packageName[len("//"):]
	}

	if strings.HasPrefix(packageName, "../") {
		return TargetID{}, errors.Wrapf(
			PackageNameNotInWorkspaceErr{
				Workspace:        workspace,
				WorkingDirectory: cwd,
				PackageName:      packageName,
			},
			"While parsing target ID %s",
			s,
		)
	}

	return TargetID{
		// trim trailing slashes from the package name
		Package: PackageName(packageName),
		Target:  TargetName(s[i+1:]),
	}, nil
}

func (tid TargetID) Freeze() {}

func (tid TargetID) String() string {
	return fmt.Sprintf("%s/%s", tid.Package, tid.Target)
}

func (tid TargetID) Truth() starlark.Bool { return starlark.Bool(true) }

func (tid TargetID) Hash() (uint32, error) {
	return JoinChecksums(
		ChecksumString(string(tid.Package)),
		ChecksumString(string(tid.Target)),
	), nil
}

func (tid TargetID) Type() string { return "Target" }

type FileGroup struct {
	Package PackageName
	Paths   []string
}

func (fg FileGroup) Freeze() {}

func (fg FileGroup) String() string {
	return fmt.Sprintf("%s:[%s]", fg.Package, strings.Join(fg.Paths, ", "))
}

func (fg FileGroup) Type() string { return "FileGroup" }

func (fg FileGroup) Truth() starlark.Bool { return starlark.Bool(true) }

func (fg FileGroup) Hash() (uint32, error) {
	checksums := make([]uint32, len(fg.Paths)+1)
	checksums[0] = ChecksumString(string(fg.Package))
	for i, path := range fg.Paths {
		checksums[i+1] = ChecksumString(path)
	}
	return JoinChecksums(checksums...), nil
}

type TargetName string

type PackageName string

type Int int64

type String string

type Bool bool

type Field struct {
	Key   string
	Value Input
}

type Object []Field

type Array []Input

type Input interface{ input() }

func (tid TargetID) input() {}
func (fg FileGroup) input() {}
func (i Int) input()        {}
func (s String) input()     {}
func (b Bool) input()       {}
func (o Object) input()     {}
func (a Array) input()      {}

type Target struct {
	ID          TargetID
	Inputs      Object
	BuilderType BuilderType
	BuilderArgs Object
}

type FrozenTargetID struct {
	Package  PackageName
	Target   TargetName
	Checksum uint32
}

func (ftid FrozenTargetID) String() string {
	return fmt.Sprintf("%s:%s@%d", ftid.Package, ftid.Target, ftid.Checksum)
}

func (ftid FrozenTargetID) ArtifactID() ArtifactID {
	return ArtifactID(ftid)
}

type FrozenField struct {
	Key   string
	Value FrozenInput
}

type TypeErr struct {
	Wanted string
	Got    string
}

func (err TypeErr) Error() string {
	return fmt.Sprintf("TypeError: expected %s, found %s", err.Wanted, err.Got)
}

func NewTypeErr(wanted string, v interface{}) TypeErr {
	return TypeErr{Wanted: wanted, Got: fmt.Sprintf("%T", v)}
}

type FrozenObject []FrozenField

type KeyNotFoundErr string

func (err KeyNotFoundErr) Error() string {
	return fmt.Sprintf("Key not found: %s", string(err))
}

func (fo FrozenObject) Get(key string) (FrozenInput, error) {
	for _, field := range fo {
		if field.Key == key {
			return field.Value, nil
		}
	}
	return nil, KeyNotFoundErr(key)
}

func (fo FrozenObject) GetString(key string) (String, error) {
	v, err := fo.Get(key)
	if err != nil {
		return "", err
	}
	if s, ok := v.(String); ok {
		return s, nil
	}
	return "", errors.Wrapf(NewTypeErr("String", v), "Key '%s'", key)
}

type FrozenArray []FrozenInput

func (fa FrozenArray) ForEach(f func(i int, elt FrozenInput) error) error {
	for i, elt := range fa {
		if err := f(i, elt); err != nil {
			return err
		}
	}

	return nil
}

func (fa FrozenArray) StringSlice() ([]string, error) {
	output := make([]string, len(fa))
	for i, v := range fa {
		if s, ok := v.(String); ok {
			output[i] = string(s)
			continue
		}
		return nil, errors.Wrapf(NewTypeErr("String", v), "Index %d", i)
	}
	return output, nil
}

func (fa FrozenArray) GetString(i int) (String, error) {
	if s, ok := fa[i].(String); ok {
		return s, nil
	}
	return "", NewTypeErr("String", fa[i])
}

type ArtifactID FrozenTargetID

func (aid ArtifactID) String() string {
	if aid.Target == "" {
		return fmt.Sprintf("//%s@%d", aid.Package, aid.Checksum)
	}
	return fmt.Sprintf("//%s:%s@%d", aid.Package, aid.Target, aid.Checksum)
}

func (aid ArtifactID) checksum() uint32 { return aid.Checksum }

func (i Int) checksum() uint32 {
	var buf [8]byte
	binary.PutVarint(buf[:len(buf)], int64(i))
	return adler32.Checksum(buf[:len(buf)])
}

func (s String) checksum() uint32 { return ChecksumString(string(s)) }

func (b Bool) checksum() uint32 {
	if bool(b) {
		return ChecksumBytes([]byte{0})
	}
	return ChecksumBytes([]byte{1})
}

func (fo FrozenObject) checksum() uint32 {
	checksums := make([]uint32, len(fo)*2)
	for i, field := range fo {
		checksums[i*2] = ChecksumString(field.Key)
		checksums[i*2+1] = field.Value.checksum()
	}
	return JoinChecksums(checksums...)
}

func (fa FrozenArray) checksum() uint32 {
	checksums := make([]uint32, len(fa))
	for i, elt := range fa {
		checksums[i] = elt.checksum()
	}
	return JoinChecksums(checksums...)
}

type FrozenInput interface {
	frozenInput()
	checksum() uint32
}

func (aid ArtifactID) frozenInput()  {}
func (i Int) frozenInput()           {}
func (s String) frozenInput()        {}
func (b Bool) frozenInput()          {}
func (fo FrozenObject) frozenInput() {}
func (fa FrozenArray) frozenInput()  {}

type FrozenTarget struct {
	ID          FrozenTargetID
	Inputs      FrozenObject
	BuilderType BuilderType
	BuilderArgs FrozenObject
}

type BuilderType string

type BuildScript func(dag DAG, cache Cache, stdout, stderr io.Writer) error

type Plugin struct {
	Type    BuilderType
	Factory func(args FrozenObject) (BuildScript, error)
}

type DAG struct {
	FrozenTarget
	Dependencies []DAG
}
