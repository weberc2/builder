# This probably won't work if there are dependencies on local packages since
# we're using modules; will need to put local dependencies in a GOPATH somehow.

def go_binary(name, sources):
    mktarget(
        name = name,
        args = {
            "modfile": file("go.mod"),
            "sumfile": file("go.sum"),
            "sources": sources,
        },
        type = "golang",
    )