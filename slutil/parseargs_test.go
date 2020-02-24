package slutil

import (
	"testing"

	"go.starlark.net/starlark"
)

func TestParseArgs_NoArgsOrKwargs(t *testing.T) {
	if err := ParseArgs(
		"foo",
		Args{},
		ArgsSpec{},
	); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
}

func TestParseArgs_SinglePosArg(t *testing.T) {
	var bar string
	if err := ParseArgs(
		"foo",
		Args{Pos: starlark.Tuple{starlark.String("bar")}},
		ArgsSpec{
			PosSpecs: []PosSpec{
				PosSpec{
					Keyword: "bar",
					Value:   ParseString(&bar),
				},
			},
		},
	); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if bar != "bar" {
		t.Fatalf("Wanted 'bar', got '%s'", bar)
	}
}

func TestParseArgs_SinglePosArgAssignedByKeyword(t *testing.T) {
	var bar string
	if err := ParseArgs(
		"foo",
		Args{
			Kw: []starlark.Tuple{{
				starlark.String("bar"), // Kw
				starlark.String("bar"), // val
			}},
		},
		ArgsSpec{
			PosSpecs: []PosSpec{{
				Keyword: "bar",
				Value:   ParseString(&bar),
			}},
		},
	); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if bar != "bar" {
		t.Fatalf("Wanted 'bar', got '%s'", bar)
	}
}

func TestParseArgs_SingleKwArgAssignedByPos(t *testing.T) {
	var bar string
	if err := ParseArgs(
		"foo",
		Args{Pos: starlark.Tuple{starlark.String("bar")}},
		ArgsSpec{
			KwSpecs: []KwSpec{{
				Keyword: "bar",
				Value:   ParseString(&bar),
				Default: starlark.String("qux"),
			}},
		},
	); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if bar != "bar" {
		t.Fatalf("Wanted 'bar', got '%s'", bar)
	}
}

func TestParseArgs_ExtraPosArgProvided(t *testing.T) {
	expectedErr := WrongPosArgCountErr{
		Fn:             "foo",
		TakesPosArgs:   0,
		TakesTotalArgs: 0,
		GivenPosArgs:   1,
	}

	if err := ParseArgs(
		"foo",
		Args{Pos: starlark.Tuple{starlark.String("bar")}},
		ArgsSpec{},
	); err != (expectedErr) {
		t.Fatalf("Expected err '%v'; got '%v'", expectedErr, err)
	}
}

func TestParseArgs_MissingPosArg(t *testing.T) {
	expectedErr := MissingRequiredArgumentsErr{
		Fn:          "foo",
		MissingHead: "bar",
	}
	var bar string
	err := ParseArgs(
		"foo",
		Args{},
		ArgsSpec{
			PosSpecs: []PosSpec{{
				Keyword: "bar",
				Value:   ParseString(&bar),
			}},
		},
	)
	if err != nil {
		if e, ok := err.(MissingRequiredArgumentsErr); ok {
			if e.Fn == expectedErr.Fn && e.MissingHead == expectedErr.MissingHead {
				if len(e.MissingTail) == len(expectedErr.MissingTail) {
					for i, arg := range e.MissingTail {
						if arg != expectedErr.MissingTail[i] {
							goto ERROR
						}
					}
					return
				}
			}
		}
	}
ERROR:
	t.Fatalf("Expected err '%v'; got '%v'", expectedErr, err)
}

func TestParseArgs_RepeatedKwarg(t *testing.T) {
	expectedErr := MultipleValuesForArgErr{Fn: "foo", Arg: "bar"}
	var bar string
	if err := ParseArgs(
		"foo",
		Args{
			Kw: []starlark.Tuple{
				{starlark.String("bar"), starlark.String("bar")},
				{starlark.String("bar"), starlark.String("bar")},
			},
		},
		ArgsSpec{
			KwSpecs: []KwSpec{{
				Keyword: "bar",
				Value:   ParseString(&bar),
				Default: starlark.String("qux"),
			}},
		},
	); err != expectedErr {
		t.Fatalf("Expected err '%v'; got '%v'", expectedErr, err)
	}
}

func TestParseArgs_MultipleValuesForSamePosArg(t *testing.T) {
	expectedErr := MultipleValuesForArgErr{Fn: "foo", Arg: "bar"}
	var bar string
	if err := ParseArgs(
		"foo",
		Args{
			Pos: starlark.Tuple{starlark.String("bar")},
			Kw: []starlark.Tuple{{
				starlark.String("bar"),
				starlark.String("bar"),
			}},
		},
		ArgsSpec{
			PosSpecs: []PosSpec{{Keyword: "bar", Value: ParseString(&bar)}},
		},
	); err != expectedErr {
		t.Fatalf("Expected err '%v'; got '%v'", expectedErr, err)
	}
}

func TestParseArgs_MultipleValuesForSameKwArg(t *testing.T) {
	expectedErr := MultipleValuesForArgErr{Fn: "foo", Arg: "bar"}
	var bar string
	if err := ParseArgs(
		"foo",
		Args{
			Pos: starlark.Tuple{starlark.String("bar")},
			Kw: []starlark.Tuple{{
				starlark.String("bar"),
				starlark.String("bar"),
			}},
		},
		ArgsSpec{
			KwSpecs: []KwSpec{{
				Keyword: "bar",
				Value:   ParseString(&bar),
				Default: starlark.String("qux"),
			}},
		},
	); err != expectedErr {
		t.Fatalf("Expected err '%v'; got '%v'", expectedErr, err)
	}
}

func TestParseArgs_UnexpectedKwarg(t *testing.T) {
	expectedErr := UnexpectedKeywordArgErr{Fn: "foo", Arg: "bar"}
	if err := ParseArgs(
		"foo",
		Args{
			Kw: []starlark.Tuple{{
				starlark.String("bar"),
				starlark.String("bar"),
			}},
		},
		ArgsSpec{},
	); err != expectedErr {
		t.Fatalf("Expected err '%v'; got '%v'", expectedErr, err)
	}
}
