load("go", "go_binary", "go_library")
load("git", "git_clone")

errors = go_library(
    name = "errors",
    package_name = "github.com/pkg/errors",
    sources = git_clone(
        name = "errors_sources",
        repo = "https://github.com/pkg/errors",
        # v0.8.1 (most recent release)
        sha = "ba968bfe8b2f7e042a574c888954fccecfa385b4",
    ),
)

go_binary(
    name = "bin",
    package_name = "github.com/weberc2/builder/examples/golang/compiledbinary",
    sources = glob("main.go"),
    dependencies = [
        reftarget("../compiledlibrary/deps:lib"),
    ],
)