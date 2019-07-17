# This probably won't work if there are dependencies on local packages since
# we're using modules; will need to put local dependencies in a GOPATH somehow.

def go_binary(name, package_name, sources, local_dependencies = None):
    args = {
        "package_name": package_name,
        "modfile": file("go.mod"),
        "sumfile": file("go.sum"),
        "sources": sources,
    }
    if local_dependencies != None:
        args["local_dependencies"] = local_dependencies
    return mktarget(name = name, args = args, type = "go_binary")

# Do a `go build` on the sources (just to validate that they compile). Write an
# artifact that contains the information required to do update the final
# `go.mod` for any `go_binary`s that depend on this per
# https://stackoverflow.com/a/52330233/483347. The updates to the final
# `go_binary`'s `go.mod` should include not just the information for this lib,
# but also for any transitive `go_library`s that this depends on.
# should cover this library as well as its local dependencies.
def go_library(name, sources, local_dependencies):
    """Declare a go_library target.

    Args:
        name: The name of the target
        sources: The sources for the library.
        local_dependencies: Other `go_library` targets that this depends on.
    """
    return mktarget(
        name = name,
        args = {
            "modfile": file("go.mod"),
            "sumfile": file("go.sum"),
            "sources": sources,
            "local_dependencies": local_dependencies,
        },
        type = "go_library",
    )