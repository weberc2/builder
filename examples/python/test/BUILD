load("std/python", "pytest")
load("examples/python/sourcelibrary", sourcelibrary = "lib")

test = pytest(
    name = "test",
    sources = glob("sourcelibrary_test.py"),
    dependencies = [ sourcelibrary ],
)