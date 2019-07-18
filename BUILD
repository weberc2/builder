load("go", "go_binary")

go_binary(
    name = "builder",
    package_name = "github.com/weberc2/builder",
    sources = glob(
        "go.sum",
        "go.mod",
        "main.go",
        "plugins/**/*.go",
        "core/**/*.go",
    ),
)