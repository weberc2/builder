load("py", pypi="py_pypi_library")

certifi = pypi(name = "certifi")
chardet = pypi(name = "chardet")
idna = pypi(name = "idna")
urllib3 = pypi(name = "urllib3")
requests = pypi(
    name = "requests",
    dependencies = [certifi, chardet, idna, urllib3],
)