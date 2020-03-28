load("std/golang", "go_module")

builder = go_module(
    name = "builder",
    sources = glob(
        "core/*.go",
        "buildutil/*.go",
        "plugins/python/*.go",
        "plugins/bash/*.go",
        "plugins/golang/*.go",
        "plugins/command/*.go",
        "slutil/*.go",
        "main.go",
        "go.mod",
        "go.sum",
    ),
)