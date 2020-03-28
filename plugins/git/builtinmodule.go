package git

const BuiltinModule = `
def git_clone(name, repo, sha = "master"):
	return mktarget(
		name = name,
		type = "git_clone",
		args = {"repo": repo, "sha": sha},
	)
`
