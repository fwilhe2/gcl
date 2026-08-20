# gcl

`gcl` is a `git clone` wrapper that puts every repository where you can find it
again: a predictable `<base-dir>/<host>/<owner>/<repo>` layout.

```sh
gcl https://github.com/fwilhe2/gcl
# → ~/code/github.com/fwilhe2/gcl
```

It can also clone *every* repository of a GitHub organization, a GitHub user,
or a GitLab group in one go, and it prints the clone path on stdout so you can
`cd` straight into it.

## Install

Download a binary for your platform from the
[latest release](https://github.com/fwilhe2/gcl/releases/latest) and put it on
your `PATH`:

```sh
tar -xzf gcl_Linux_x86_64.tar.gz
sudo install gcl /usr/local/bin
```

Archives are published for Linux, macOS, and Windows (x86_64, arm64, i386).

Or build from source (Go 1.24+):

```sh
go install github.com/fwilhe2/gcl@latest
```

## Usage

```text
gcl <repository-url|owner-url> [flags]

Flags:
      --all               clone all repositories of the given owner, even for URLs
                          with multiple path segments (e.g. GitLab subgroups)
      --base-dir string   base directory for cloned repositories
  -v, --version           version for gcl
```

### Cloning a single repository

```sh
gcl https://github.com/fwilhe2/gcl
```

HTTPS URLs, `ssh://` URLs, and scp-like SSH URLs all work, and a trailing
`.git` or `/` is stripped — so these end up in the same directory:

```sh
gcl https://github.com/fwilhe2/gcl
gcl ssh://git@github.com/fwilhe2/gcl.git
gcl git@github.com:fwilhe2/gcl.git
```

Submodules are cloned recursively. If the target directory already exists,
`gcl` leaves it alone and just prints its path — so re-running is safe.

### Jumping into the clone

Progress and status messages go to stderr, the clone path goes to stdout:

```sh
cd "$(gcl https://github.com/fwilhe2/gcl)"
```

A handy shell function:

```sh
gclcd() { cd "$(gcl "$1")"; }
```

### Choosing where things land

By default everything is cloned below `~/code`. Change it permanently with
`GCL_BASE_DIR`, or per invocation with `--base-dir`:

```sh
export GCL_BASE_DIR=~/src              # ~/src/github.com/fwilhe2/gcl
gcl --base-dir ~/work https://github.com/fwilhe2/gcl
```

### Cloning everything of an organization, user, or group

Pass an owner URL — a URL with a single path segment — and `gcl` clones all of
its repositories into the same layout:

```sh
gcl https://github.com/my-org
gcl https://gitlab.com/my-group
```

Already cloned repositories are skipped, so an interrupted run can simply be
restarted. Failures are reported per repository and don't abort the rest of the
run. GitLab group listings include all nested subgroups, and the nested layout
is preserved on disk.

A GitLab subgroup URL has more than one path segment and would otherwise look
like a repository, so force owner mode with `--all`:

```sh
gcl --all https://gitlab.com/my-group/my-subgroup
```

Supported forges are `github.com` and `gitlab.com`. For self-hosted GitLab
instances, list their hostnames in `GCL_GITLAB_HOSTS` — the API is always at
`/api/v4` on the same host, so no further configuration is needed:

```sh
GCL_GITLAB_HOSTS=git.example.com,gitlab.internal gcl https://git.example.com/my-group
```

Self-hosted GitHub (GitHub Enterprise Server) instances don't expose their API
at a fixed path relative to the host, so list them in `GCL_GITHUB_HOSTS` as
`host=apiBaseURL` pairs instead:

```sh
GCL_GITHUB_HOSTS=github.example.com=https://github.example.com/api/v3 gcl https://github.example.com/my-org
```

### Authentication

For cloning over HTTP(S), `gcl` asks your configured git credential helper
(`git credential fill`) for credentials — whatever already works for
`git clone` works here. It never prompts interactively: if no helper has a
matching credential, the clone proceeds unauthenticated. SSH URLs use your
normal SSH setup.

Listing an owner's repositories goes through the forge API instead. Set
`GITHUB_TOKEN` or `GITLAB_TOKEN` for higher rate limits and access to private
repositories.

### Configuration reference

Every setting below can be set with an env var, or, if you'd rather not repeat
yourself in every shell, with the matching key in a JSON config file. Both can
be used together — the env var wins if a setting is present in both.

| Env var | Config file key | Purpose | Default |
| --- | --- | --- | --- |
| `GCL_BASE_DIR` | `base_dir` | Base directory for clones (`~` is expanded) | `~/code` |
| `GCL_GITLAB_HOSTS` | `gitlab_hosts` | Hostnames to treat as GitLab | — |
| `GCL_GITHUB_HOSTS` | `github_hosts` | Hosts to treat as GitHub, with their API base URL | — |
| `GITHUB_TOKEN` | `github_token` | Auth for the GitHub repository listing API | — |
| `GITLAB_TOKEN` | `gitlab_token` | Auth for the GitLab repository listing API | — |
| `GCL_CONFIG` | — | Path to the config file itself | see below |

#### Config file location

`gcl` reads `$GCL_CONFIG` if set, otherwise the config file at the OS's
standard per-user config location:

| OS | Default path |
| --- | --- |
| Linux | `~/.config/gcl/config.json` (or `$XDG_CONFIG_HOME/gcl/config.json` if set) |
| macOS | `~/Library/Application Support/gcl/config.json` |
| Windows | `%AppData%\gcl\config.json` (typically `C:\Users\<you>\AppData\Roaming\gcl\config.json`) |

The file is entirely optional, and a missing file is not an error.

#### Full example

```json
{
  "base_dir": "/home/alice/code",
  "github_token": "ghp_ExampleGitHubToken1234567890",
  "gitlab_token": "glpat-ExampleGitLabToken1234567",
  "github_hosts": {
    "github.example.com": { "api_base": "https://github.example.com/api/v3" },
    "ghe.corp.internal": { "api_base": "https://ghe.corp.internal/api/v3", "token": "ghp_ExampleEnterpriseToken1234567890" }
  },
  "gitlab_hosts": ["git.example.com", "gitlab.internal"]
}
```

Each entry in `github_hosts` may set its own `token`, used instead of
`github_token`/`GITHUB_TOKEN` for that instance.

## Development

```sh
make            # format, build, test
make build      # go build -o gcl .
make test       # go test ./...
make format     # gofumpt
make install    # install ./gcl to /usr/local/bin
make update     # go get -u && go mod tidy
```

`gcl --version` reports the version, commit, tree state, build time, and Go
toolchain. Only the release pipeline stamps a real version, so local builds
identify themselves as development builds:

```console
$ make build && ./gcl --version
gcl v0.0.3-0.20260731160415-12df1a10e885+dirty (development build, not a release)
commit: 12df1a10e88538af4b85a3be5409964a7a00286e (dirty git tree)
built:  2026-07-31T16:04:15Z
go:     go1.24.4 linux/amd64
```

Version, commit, and tree state fall back to the build info the Go toolchain
embeds, which is why a local build shows a pseudo-version derived from the last
tag — the `development build` marker is what distinguishes it from a release.

Pass `VERSION` to mimic a release build locally:

```sh
make build VERSION=1.2.3
```

## Releasing

Releases are cut by manually running the
[Release workflow](.github/workflows/release.yml) — **not** by pushing a tag by
hand. Trigger it from the Actions tab, or with the GitHub CLI:

```sh
gh workflow run Release -f component=patch   # or: minor, major
```

The workflow then:

1. runs [`fwilhe2/bump-version`](https://github.com/fwilhe2/bump-version) to
   derive the next version from the latest tag and the chosen `component`
   (defaults to `patch`),
2. creates and pushes that tag,
3. runs [GoReleaser](https://goreleaser.com) (`.goreleaser.yaml`) to build the
   cross-platform archives, stamp version/commit/date/tree state into the
   binary via `-ldflags`, generate the changelog, and publish the GitHub
   release.

Every push and pull request to `main` additionally runs the
[Build workflow](.github/workflows/go.yaml) (`go build` + `go test`).

## License

[Apache License 2.0](LICENSE)
