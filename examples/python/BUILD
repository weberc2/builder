load("examples/python/sourcebinary", "sourcebinary")
load("examples/python/test", "test")

all = mktarget(
    name = "all",
    type = "noop",
    args = {"dependencies": [sourcebinary, test]},
)