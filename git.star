def git_clone(name, repo, sha):
    return mktarget(
        name = name,
        args = {"repo": repo, "sha": sha},
        type = "git_clone",
    )