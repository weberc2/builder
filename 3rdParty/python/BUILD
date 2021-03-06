load("std/python", "pypi")

atomicwrites = pypi(name = "atomicwrites")
six = pypi(name = "six")
more_itertools = pypi(name = "more-itertools", dependencies = [ six ])
zipp = pypi(name = "zipp", dependencies = [ more_itertools ])
importlib_metadata = pypi(name = "importlib_metadata", dependencies = [ zipp ])
pluggy = pypi(name = "pluggy", dependencies = [ importlib_metadata ])
py = pypi(name = "py")
wcwidth = pypi(name = "wcwidth")
packaging = pypi(name = "packaging")
certifi = pypi(name = "certifi")
chardet = pypi(name = "chardet")
idna = pypi(name = "idna")
urllib3 = pypi(name = "urllib3")
requests = pypi(
    name = "requests",
    dependencies = [certifi, chardet, idna, urllib3],
)

psycopg2_binary = pypi(name = "psycopg2-binary")
aiopg = pypi(name = "aiopg", dependencies = [psycopg2_binary])
aiotask_context = pypi(name = "aiotask_context")

python_dateutil = pypi(name = "python-dateutil", dependencies = [six])
pytz = pypi(name = "pytz")
numpy = pypi(name = "numpy", constraint = ">=1.16.4")
pandas = pypi(name = "pandas", dependencies = [python_dateutil, pytz, numpy])

aenum = pypi(name = "aenum")
attrs = pypi(name = "attrs")
newrelic = pypi(name = "newrelic")
arrow = pypi(
    name = "arrow",
    constraint = "==0.12.1",
    dependencies = [python_dateutil],
)
sentry_sdk = pypi(name = "sentry-sdk", dependencies = [certifi, urllib3])
async_timeout = pypi(name = "async-timeout")
hiredis = pypi(name = "hiredis")
aioredis = pypi(name = "aioredis")
sentinels = pypi(name = "sentinels")
sqlalchemy = pypi(name = "sqlalchemy")
slugify = pypi(name = "slugify")
astrolabe = pypi(name = "astrolabe")
multidict = pypi(name = "multidict")
yarl = pypi(name = "yarl", dependencies = [idna])
idna_ssl = pypi(name = "idna-ssl")
typing_extensions = pypi(
    name = "typing_extensions",
    # >=3.7.4 - black==19.10.b0
    constraint = ">=3.7.4",
)
aiohttp = pypi(
    name = "aiohttp",
    # >=3.3.2 - black==19.10.b0
    constraint = "==3.4.4", # Not sure where this 3.4.4 came from
    dependencies = [
        attrs,
        chardet,
        multidict,
        async_timeout,
        yarl,
        idna_ssl,
        typing_extensions,
    ],
)
isoweek = pypi(name = "isoweek")

decorator = pypi(name = "decorator")
cfuzzyset = pypi(name = "cfuzzyset")
pycparser = pypi(name = "pycparser")
cffi = pypi(name = "cffi", dependencies = [pycparser])
bcrypt = pypi(name = "bcrypt", dependencies = [six, cffi])
asn1crypto = pypi(name = "asn1crypto")
cryptography = pypi(name = "cryptography", dependencies = [asn1crypto, cffi])
pynacl = pypi(name = "pynacl")
paramiko = pypi(
    name = "paramiko",
    dependencies = [bcrypt, cryptography, pynacl],
)
sshtunnel = pypi(name = "sshtunnel", dependencies = [paramiko])
jmespath = pypi(name = "jmespath")
docutils = pypi(name = "docutils")
botocore = pypi(
    name = "botocore",
    dependencies = [jmespath, docutils, urllib3, python_dateutil],
)
s3transfer = pypi(name = "s3transfer")
boto3 = pypi(name = "boto3", dependencies = [botocore, s3transfer])
scipy = pypi(name = "scipy")
boto = pypi(name = "boto")
smart_open = pypi(name = "smart_open", dependencies = [boto, requests])
gensim = pypi(name = "gensim", dependencies = [numpy, scipy, smart_open])
markupsafe = pypi(name = "MarkupSafe")
jinja2 = pypi(name = "jinja2", dependencies = [markupsafe])
zope_interface = pypi(name = "zope.interface")
datetime = pypi(name = "datetime", dependencies = [zope_interface])
xmltodict = pypi(name = "xmltodict")
werkzeug = pypi(name = "werkzeug")
pyyaml = pypi(name = "PyYAML")
ecdsa = pypi(name = "ecdsa")
future = pypi(name = "future")
pyasn1 = pypi(name = "pyasn1")
rsa = pypi(name = "rsa", dependencies = [pyasn1])
python_jose = pypi(name = "python-jose", dependencies = [ecdsa, future, rsa])
mock = pypi(name = "mock")
websocket_client = pypi(name = "websocket-client")
docker = pypi(name = "docker", dependencies = [ websocket_client ])
jsondiff = pypi(name = "jsondiff")
jsonpickle = pypi(name = "jsonpickle")
wrapt = pypi(name = "wrapt")
aws_xray_sdk = pypi(name = "aws-xray-sdk", dependencies = [jsonpickle, wrapt])
responses = pypi(name = "responses")
aws_sam_translator = pypi(name = "aws-sam-translator")
jsonpointer = pypi(name = "jsonpointer")
jsonpatch = pypi(name = "jsonpatch", dependencies = [jsonpointer])
pyrsistent = pypi(name = "pyrsistent")
jsonschema = pypi(name = "jsonschema", dependencies = [attrs, pyrsistent])
setuptools = pypi(name = "setuptools")
cfn_lint = pypi(
    name = "cfn-lint",
    dependencies = [aws_sam_translator, jsonpatch, jsonschema, setuptools],
)
sshpubkeys = pypi(name = "sshpubkeys")
moto = pypi(
    name = "moto",
    dependencies = [
        jinja2,
        boto,
        boto3,
        botocore,
        cryptography,
        datetime,
        requests,
        xmltodict,
        six,
        werkzeug,
        pyyaml,
        pytz,
        python_dateutil,
        python_jose,
        mock,
        docker,
        jsondiff,
        aws_xray_sdk,
        responses,
        idna,
        cfn_lint,
        sshpubkeys,
    ],
)
python_levenshtein = pypi(
    name = "python-levenshtein",
    dependencies = [setuptools],
)
texttable = pypi(name = "texttable")
fuzzyset = pypi(
    name = "fuzzyset",
    dependencies = [texttable, python_levenshtein],
)
terminaltables = pypi(name = "terminaltables")
inflect = pypi(name = "inflect")
backoff = pypi(name = "backoff")
semver = pypi(name = "semver")
pyrfc3339 = pypi(name = "pyrfc3339", dependencies = [pytz])
ldclient = pypi(
    name = "ldclient-py",
    dependencies = [backoff, semver, pyrfc3339],
)
docopt = pypi(name = "docopt")
mandrill = pypi(name = "mandrill", dependencies = [requests, docopt])
alooma = pypi(name = "alooma", dependencies = [urllib3, requests])
auth0 = pypi(
    name = "auth0",
    pypi_name = "auth0-python",
    dependencies = [requests],
)
jwt = pypi(name = "jwt", pypi_name = "PyJWT")
freezegun = pypi(name = "freezegun", dependencies = [six, python_dateutil])
appdirs = pypi(name = "appdirs")
click = pypi(
    name = "click",
    # >=6.5 - black==19.10.b0
    constraint = ">=6.5",
)
toml = pypi(
    name = "toml",
    # >=0.9.4 - black==19.10.b0
    constraint = ">=0.9.4",
)
regex = pypi(
    name = "regex",
    # >=2019.8 - black==19.10.b0
    constraint = ">=2019.8",
)
typed_ast = pypi(
    name = "typed_ast",
    # ==1.4.0 - black==19.10.b0
    constraint = "==1.4.0",
)
aiohttp_cors = pypi(name = "aiohttp-cors")
dataclasses = pypi(
    name = "dataclasses",
    # >=0.6 - black==19.10.b0
    constraint = ">=0.6",
)
pathspec = pypi(
    name = "pathspec",
    # >=0.6 - black==19.10.b0
    constraint = ">=0.6",
)
mypy_extensions = pypi(
    name = "mypy_extensions",
    # >=0.4.3 - black==19.10.b0
    constraint = ">=0.4.3",
)

# https://github.com/psf/black/blob/19.10.b0/Pipfile
# aiohttp = ">=3.3.2"
# attrs = ">=18.1.0"
# click = ">=6.5"
# appdirs = "*"
# toml = ">=0.9.4"
# black = {path = ".",extras = ["d"],editable = true}
# aiohttp-cors = "*"
# typed-ast = "==1.4.0"
# regex = ">=2019.8"
# pathspec = ">=0.6"
black = pypi(
    name = "black",
    constraint = "==19.10.b0", # latest version as of 2019-11-06
    dependencies = [
        aiohttp,
        aiohttp_cors,
        appdirs,
        attrs,
        click,
        dataclasses,
        mypy_extensions,
        regex,
        pathspec,
        toml,
        typed_ast,
        typing_extensions,
    ],
)
