# NOTES

## Considerations

* We will not use GOPATH-aware tools to build, but rather we'll invoke
  `go tool compile` directly to avoid the tedium of making GOPATH-compatible
  directory structures (in the cache or elsewhere) but also to avoid using
  tools that might respect a go.mod and accidentally bring in dependencies from
  elsewhere.

## Approach

* Go has SourceLibrary artifacts which are a directory of Go files. These don't
  need to be GOPATH-compatible because we'll invoke `go tool compile` on them
  directly to build. By making
  source code lives under a src/${PACKAGE_NAME}/ directory).
* Go also has a CompiledLibrary target type which takes a SourceLibrary
  target and produces an archive (a `.a` file).
* Go has RemoteLibrary targets which take a github URL or similar and produce a
  SourceLibrary artifact.

## Open Questions

* How are editors' GOPATHs set?
  * Plugin allows for the GOPATH to be echoed?
    * Perhaps a custom target whose artifact is a script that echos the GOPATH?