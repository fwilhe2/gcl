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
