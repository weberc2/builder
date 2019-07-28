load("go", "go_library")

go_library(
    name = "lib",
    package_name = "github.com/weberc2/builder/examples/golang/compiledlibrary/deps",
    sources = glob("*.go"),
    dependencies = [reftarget("../nodeps:lib")],
)