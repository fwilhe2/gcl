# gcl
git clone wrapper with automatic directory layout

`gcl` clones repositories into a predictable host/owner/repo directory layout.

## Usage

```sh
gcl https://github.com/fwilhe2/gcl
```

By default, repositories are cloned below `~/code`:

```text
~/code/github.com/fwilhe2/gcl
```

Set `GCL_BASE_DIR` to use a different default base directory:

```sh
GCL_BASE_DIR=~/src gcl https://github.com/fwilhe2/gcl
```

Use `--base-dir` for a one-off override:

```sh
gcl --base-dir ~/work https://github.com/fwilhe2/gcl
```

Supported URL formats include HTTPS URLs, `ssh://` URLs, and scp-like SSH URLs:

```sh
gcl https://github.com/fwilhe2/gcl
gcl ssh://git@github.com/fwilhe2/gcl.git
gcl git@github.com:fwilhe2/gcl.git
```

## Releases

Releases are built by GitHub Actions when a `v*` tag is pushed:

```sh
git tag v1.2.3
git push origin v1.2.3
```

The release workflow uses GoReleaser and bakes the tag version into Cobra's
`--version` output. For local builds, pass `VERSION` to `make build`:

```sh
make build VERSION=1.2.3
./gcl --version
```
