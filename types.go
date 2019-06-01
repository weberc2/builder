package main

import (
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
)

type TargetID struct {
	Package PackageName
	Target  TargetName
}

func parseTargetID(s string) (TargetID, error) {
	i := strings.Index(string(s), ":")
	if i < 1 { // must have a colon; can't be first character
		return TargetID{}, fmt.Errorf("ValueError: Invalid target ref")
	}

	return TargetID{
		Package: PackageName(s[:i]),
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

type SourcePath struct {
	Package  PackageName
	Target   TargetName
	FilePath string
}

func (sp SourcePath) ArtifactID(checksum uint32) ArtifactID {
	return ArtifactID{
		FrozenTargetID: FrozenTargetID{
			Package:  sp.Package,
			Target:   sp.Target,
			Checksum: checksum,
		},
		FilePath: sp.FilePath,
	}
}

func (sp SourcePath) Freeze() {}

func (sp SourcePath) String() string {
	return filepath.Join(string(sp.Package), string(sp.Target), sp.FilePath)
}

func (sp SourcePath) Type() string { return "SourcePath" }

func (sp SourcePath) Truth() starlark.Bool { return starlark.Bool(true) }

func (sp SourcePath) Hash() (uint32, error) {
	return JoinChecksums(
		ChecksumString(string(sp.Package)),
		ChecksumString(string(sp.Target)),
		ChecksumString(sp.FilePath),
	), nil
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

func (tid TargetID) input()  {}
func (sp SourcePath) input() {}
func (i Int) input()         {}
func (s String) input()      {}
func (b Bool) input()        {}
func (o Object) input()      {}
func (a Array) input()       {}

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
	return ArtifactID{FrozenTargetID: ftid}
}

type FrozenField struct {
	Key   string
	Value FrozenInput
}

type FrozenObject []FrozenField

type FrozenArray []FrozenInput

type ArtifactID struct {
	FrozenTargetID
	FilePath string
}

func (aid ArtifactID) String() string {
	return filepath.Join(
		string(aid.Package),
		string(aid.Target),
		string(aid.FilePath),
		fmt.Sprint(aid.Checksum),
	)
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