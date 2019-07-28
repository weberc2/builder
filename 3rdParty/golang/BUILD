load("go", "go_binary")
load("git", "git_clone")

sys_unix = go_binary(
    name = "sys-unix",
    package_name = "golang.org/x/sys",
    sources = git_clone(
        name = "sys-unix_sources",
        repo = "https://github.com/golang/sys",
        sha = "fc99dfbffb4e5ed5758a37e31dd861afe285406b",
    ),
    directory = "unix",
)