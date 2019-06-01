load("noop", "noop")

noop(name = "baz", hello="world")

noop(
    name = "bar",
    hello = "world",
    sources = [file("asdf")],
    dependencies = [ reftarget("foo:baz")],
    baz = "qux",
)