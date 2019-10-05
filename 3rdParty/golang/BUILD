load("go", "go_module")
load("git", "git_clone")

doublestar = go_module(
    name = "doublestar",
    module_name = "github.com/bmatcuk/doublestar",
    sources = git_clone(
        name = "doublestar_sources",
        repo = "https://github.com/bmatcuk/doublestar",
        sha = "v1.1.1",
    ),
)

go_isatty = go_module(
    name = "go-isatty",
    module_name = "github.com/mattn/go-isatty",
    sources = git_clone(
        name = "go-isatty_sources",
        repo = "https://github.com/mattn/go-isatty",
        sha = "v0.0.8",
    ),
)

go_colorable = go_module(
    name = "go-colorable",
    module_name = "github.com/mattn/go-colorable",
    sources = git_clone(
        name = "go-colorable_sources",
        repo = "https://github.com/mattn/go-colorable",
        sha = "v0.1.2",
    ),
    dependencies = [go_isatty],
)

color = go_module(
    name = "color",
    module_name = "github.com/fatih/color",
    sources = git_clone(
        name = "color_sources",
        repo = "https://github.com/fatih/color",
        sha = "v1.7.0",
    ),
    dependencies = [go_colorable],
)

errors = go_module(
    name = "errors",
    module_name = "github.com/pkg/errors",
    sources = git_clone(
        name = "errors_sources",
        repo = "https://github.com/pkg/errors",
        sha = "v0.8.1",
    ),
)

starlark = go_module(
    name = "starlark",
    module_name = "go.starlark.net",
    sources = git_clone(
        name = "starlark_sources",
        repo = "https://github.com/google/starlark-go",
        sha = "30ae18b8564f6cd89b550b19dbdae92e0ec5b0a3",
    ),
    provides = ["syntax", "internal/spell", "resolve", "internal/compile", "starlark"],
)

sys = go_module(
    name = "sys",
    module_name = "golang.org/x/sys",
    sources = git_clone(
        name = "sys_sources",
        repo = "https://github.com/golang/sys",
        sha = "",
    ),
    provides = ["unix"],
)

crypto = go_module(
    name = "crypto",
    module_name = "golang.org/x/crypto",
    sources = git_clone(
        name = "crypto_sources",
        repo = "https://github.com/golang/crypto",
        sha = "",
    ),
    dependencies = [sys],
    provides = ["openpgp", "ssh", "ssh/knownhosts"],
)

net = go_module(
    name = "net",
    module_name = "golang.org/x/net",
    sources = git_clone(
        name = "net_sources",
        repo = "https://github.com/golang/net",
        sha = "",
    ),
    provides = ["context", "proxy"],
)

gods = go_module(
    name = "gods",
    module_name = "github.com/emirpasic/gods",
    sources = git_clone(
        name = "gods_sources",
        repo = "https://github.com/emirpasic/gods",
        sha = "v1.12.0",
    ),
    provides = [
        "utils",
        "containers",
        "lists/arraylist",
        "trees/binaryheap",
    ],
)

go_context = go_module(
    name = "go-context",
    module_name = "github.com/jbenet/go-context",
    sources = git_clone(
        name = "go-context_sources",
        repo = "https://github.com/jbenet/go-context",
        sha = "d14ea06fba99483203c19d92cfcd13ebe73135f4",
    ),
    provides = ["io"],
    dependencies = [net],
)

go_buffruneio = go_module(
    name = "go-buffruneio",
    module_name = "github.com/pelletier/go-buffruneio",
    sources = git_clone(
        name = "go-buffruneio_sources",
        repo = "https://github.com/pelletier/go-buffruneio",
        sha = "v0.2.0",
    ),
)

ssh_config = go_module(
    name = "ssh-config",
    module_name = "github.com/kevinburke/ssh_config",
    sources = git_clone(
        name = "ssh-config_sources",
        repo = "https://github.com/kevinburke/ssh_config",
        sha = "81db2a75821ed34e682567d48be488a1c3121088",
    ),
    dependencies = [go_buffruneio],
)

go_homedir = go_module(
    name = "go-homedir",
    module_name = "github.com/mitchellh/go-homedir",
    sources = git_clone(
        name = "go-homedir_sources",
        repo = "https://github.com/mitchellh/go-homedir",
        sha = "v1.1.0",
    ),
)

go_diff = go_module(
    name = "go-diff",
    module_name = "github.com/sergi/go-diff",
    sources = git_clone(
        name = "go-diff_sources",
        repo = "https://github.com/sergi/go-diff",
        sha = "v1.0.0",
    ),
    provides = ["diffmatchpatch"],
)

warnings = go_module(
    name = "warnings",
    module_name = "gopkg.in/warnings.v0",
    sources = git_clone(
        name = "warnings_sources",
        repo = "https://github.com/go-warnings/warnings",
        sha = "v0.1.2",
    ),
)

gcfg = go_module(
    name = "gcfg",
    module_name = "github.com/src-d/gcfg",
    sources = git_clone(
        name = "gcfg_sources",
        repo = "https://github.com/src-d/gcfg",
        sha = "1ac3a1ac202429a54835fe8408a92880156b489d",
    ),
    dependencies = [warnings],
)

ssh_agent = go_module(
    name = "ssh-agent",
    module_name = "github.com/xanzy/ssh-agent",
    sources = git_clone(
        name = "ssh-agent_sources",
        repo = "https://github.com/xanzy/ssh-agent",
        sha = "v0.2.1",
    ),
    dependencies = [crypto],
)

creack_pty = go_module(
    name = "creack-pty",
    module_name = "github.com/creack/pty",
    sources = git_clone(
        name = "creak-pty_sources",
        repo = "https://github.com/creack/pty",
        sha = "1.1.7",
    ),
)

pty = go_module(
    name = "pty",
    module_name = "github.com/kr/pty",
    sources = git_clone(
        name = "pty_sources",
        repo = "https://github.com/kr/pty",
        sha = "1.1.1",
    ),
    dependencies = [creack_pty],
)

text = go_module(
    name = "text",
    module_name = "github.com/kr/text",
    sources = git_clone(
        name = "text_sources",
        repo = "https://github.com/kr/text",
        # For some reason I get the following error if sha == v0.1.0:
        # Checking out sha v0.1.0: reference not found
        sha = "e2ffdb16a802fe2bb95e2e35ff34f0e53aeef34f",
    ),
    dependencies = [pty],
)

pretty = go_module(
    name = "pretty",
    module_name = "github.com/kr/pretty",
    sources = git_clone(
        name = "pretty_sources",
        repo = "https://github.com/kr/pretty",
        # For some reason I get the following error if sha == v0.1.0:
        # Checking out sha v0.1.0: reference not found
        sha = "73f6ac0b30a98e433b289500d779f50c1a6f0712",
    ),
    dependencies = [text],
)

check = go_module(
    name = "check",
    module_name = "gopkg.in/check.v1",
    sources = git_clone(
        name = "check_sources",
        repo = "https://github.com/go-check/check",
        sha = "788fd78401277ebd861206a03c884797c6ec5541",
    ),
    dependencies = [pretty],
)

go_billy = go_module(
    name = "go-billy",
    module_name = "gopkg.in/src-d/go-billy.v4",
    sources = git_clone(
        name = "go-billy_sources",
        repo = "https://github.com/src-d/go-billy",
        sha = "v4.3.2",
    ),
    provides = ["util", "osfs", "."],
    dependencies = [sys, check],
)

go_flags = go_module(
    name = "go-flags",
    module_name = "github.com/jessevdk/go-flags",
    sources = git_clone(
        name = "go-flags_sources",
        repo = "https://github.com/jessevdk/go-flags",
        sha = "v1.4.0",
    ),
)

go_git = go_module(
    name = "go-git",
    module_name = "gopkg.in/src-d/go-git.v4",
    sources = git_clone(
        name = "go-git_sources",
        repo = "https://github.com/src-d/go-git",
        sha = "v4.12.0",
    ),
    dependencies = [
        gods,
        go_context,
        crypto,
        ssh_config,
        go_homedir,
        go_diff,
        gcfg,
        ssh_agent,
        net,
        go_billy,
        go_buffruneio,
        sys,
        warnings,
        go_flags,
    ],
)