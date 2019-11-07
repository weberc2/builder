load("lib://py", "py_source_library")

lib = py_source_library(
    name = "lib",
    package_name = "sourcelibrary",
    sources = glob(
        "setup.py",
        "src/sourcelibrary/__init__.py",
        "src/sourcelibrary/greet.py",
    ),
    dependencies = [],
)
