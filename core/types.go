package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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

func (tid TargetID) String() string {
	return fmt.Sprintf("%s/%s", tid.Package, tid.Target)
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

func (t Target) Freeze() {}

func (t Target) String() string {
	return fmt.Sprintf("Target(%s)", t.ID)
}

func (t Target) Truth() starlark.Bool { return starlark.Bool(true) }

func (t Target) Hash() (uint32, error) {
	return t.hash(), nil
}

func (t Target) Type() string { return "Target" }

type FileGroup struct {
	Package  PackageName
	Patterns []string
}

func (fg FileGroup) Freeze() {}

func (fg FileGroup) String() string {
	return fmt.Sprintf("%s:[%s]", fg.Package, strings.Join(fg.Patterns, ", "))
}

func (fg FileGroup) Type() string { return "FileGroup" }

func (fg FileGroup) Truth() starlark.Bool { return starlark.Bool(true) }

func (fg FileGroup) Hash() (uint32, error) {
	return fg.hash(), nil
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

func (f Field) marshalJSON(buf *bytes.Buffer) error {
	data, err := json.Marshal(f.Key)
	if err != nil {
		panic(fmt.Sprintf(
			"Marshaling types.Object field key '%s': %v",
			f.Key,
			err,
		))
	}
	buf.Write(data)
	buf.WriteByte(':')
	data, err = json.Marshal(f.Value)
	if err != nil {
		return errors.Wrapf(err, "Marshaling field '%s'", f.Key)
	}
	buf.Write(data)
	return nil
}

type Object []Field

func (o Object) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf.WriteByte('{')
	if len(o) > 0 {
		if err := o[0].marshalJSON(buf); err != nil {
			return nil, err
		}
		for _, field := range o[1:] {
			buf.WriteByte(',')
			if err := field.marshalJSON(buf); err != nil {
				return nil, err
			}
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type Array []Input

type Input interface {
	input()
	hash() uint32
}

func (t Target) input() {}
func (t Target) hash() uint32 {
	return JoinChecksums(
		ChecksumString(string(t.ID.Package)),
		ChecksumString(string(t.ID.Target)),
		t.Inputs.hash(),
		ChecksumString(string(t.BuilderType)),
	)
}
func (fg FileGroup) input() {}
func (fg FileGroup) hash() uint32 {
	checksums := make([]uint32, len(fg.Patterns)+1)
	checksums[0] = ChecksumString(string(fg.Package))
	for i, pattern := range fg.Patterns {
		checksums[i+1] = ChecksumString(pattern)
	}
	return JoinChecksums(checksums...)
}
func (i Int) input() {}
func (i Int) hash() uint32 {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(i))
	return ChecksumBytes(buf[:])
}
func (s String) input()       {}
func (s String) hash() uint32 { return ChecksumString(string(s)) }
func (b Bool) input()         {}
func (b Bool) hash() uint32 {
	var i uint16
	if bool(b) {
		i = 1
	}
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], i)
	return ChecksumBytes(buf[:])
}
func (o Object) input() {}
func (o Object) hash() uint32 {
	checksums := make([]uint32, 2*len(o))
	for i, f := range o {
		checksums[2*i] = ChecksumString(f.Key)
		checksums[2*i+1] = f.Value.hash()
	}
	return JoinChecksums(checksums...)
}
func (a Array) input() {}
func (a Array) hash() uint32 {
	checksums := make([]uint32, len(a))
	for i, v := range a {
		checksums[i] = v.hash()
	}
	return JoinChecksums(checksums...)
}

type Target struct {
	ID          TargetID
	Inputs      Object
	BuilderType BuilderType
}

func (t Target) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Package string `json:"package"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Inputs  Object `json:"inputs"`
	}{
		Package: string(t.ID.Package),
		Name:    string(t.ID.Target),
		Type:    string(t.BuilderType),
		Inputs:  t.Inputs,
	})
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

type KeySpec struct {
	Key   string
	Value func(FrozenInput) error
}

func (fo FrozenObject) VisitKeys(keys ...KeySpec) error {
	for _, key := range keys {
		if err := fo.VisitKey(key.Key, key.Value); err != nil {
			return err
		}
	}
	return nil
}

func (fo FrozenObject) VisitKey(
	key string,
	f func(FrozenInput) error,
) error {
	for _, field := range fo {
		if field.Key == key {
			return errors.Wrapf(f(field.Value), "Visiting key '%s'", key)
		}
	}
	return errors.Wrapf(KeyNotFoundErr(key), "Visiting key '%s'", key)
}

func ParseString(sptr *string) func(FrozenInput) error {
	return AssertString(func(s string) error {
		*sptr = s
		return nil
	})
}

func AssertString(f func(string) error) func(FrozenInput) error {
	return func(fi FrozenInput) error {
		if s, ok := fi.(String); ok {
			return f(string(s))
		}
		return TypeErr{Wanted: "String", Got: fmt.Sprintf("%T", fi)}
	}
}

func AssertInt(f func(int) error) func(FrozenInput) error {
	return func(fi FrozenInput) error {
		if i, ok := fi.(Int); ok {
			return f(int(i))
		}
		return TypeErr{Wanted: "Int", Got: fmt.Sprintf("%T", fi)}
	}
}

func AssertArtifactID(f func(ArtifactID) error) func(FrozenInput) error {
	return func(fi FrozenInput) error {
		if aid, ok := fi.(ArtifactID); ok {
			return f(aid)
		}
		return TypeErr{Wanted: "ArtifactID", Got: fmt.Sprintf("%T", fi)}
	}
}

func ParseArtifactID(aidptr *ArtifactID) func(FrozenInput) error {
	return AssertArtifactID(func(aid ArtifactID) error {
		*aidptr = aid
		return nil
	})
}

func AssertArray(f func(FrozenArray) error) func(FrozenInput) error {
	return func(fi FrozenInput) error {
		if fa, ok := fi.(FrozenArray); ok {
			return f(fa)
		}
		return TypeErr{Wanted: "List", Got: fmt.Sprintf("%T", fi)}
	}
}

func AssertArrayOf(
	f func(FrozenInput) error,
) func(FrozenInput) error {
	return AssertArray(func(fa FrozenArray) error {
		for i, elt := range fa {
			if err := f(elt); err != nil {
				return errors.Wrapf(err, "At element %d", i)
			}
		}
		return nil
	})
}

func AssertObject(f func(FrozenObject) error) func(FrozenInput) error {
	return func(fi FrozenInput) error {
		if fo, ok := fi.(FrozenObject); ok {
			return f(fo)
		}
		return TypeErr{Wanted: "Dict", Got: fmt.Sprintf("%T", fi)}
	}
}

func AssertObjectOf(
	f func(FrozenField) error,
) func(FrozenInput) error {
	return AssertObject(func(fo FrozenObject) error {
		for _, field := range fo {
			if err := f(field); err != nil {
				return errors.Wrapf(err, "At field %s", field.Key)
			}
		}
		return nil
	})
}

// Deprecated in favor of VisitKey()
func (fo FrozenObject) Get(key string) (FrozenInput, error) {
	for _, field := range fo {
		if field.Key == key {
			return field.Value, nil
		}
	}
	return nil, KeyNotFoundErr(key)
}

// Deprecated in favor of VisitKey(..., ParseString())
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
}

type BuilderType string

type BuildScript func(dag DAG, cache Cache, stdout, stderr io.Writer) error

type Plugin struct {
	Type        BuilderType
	BuildScript BuildScript
}

type DAG struct {
	FrozenTarget
	Dependencies []DAG
}
