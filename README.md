# teamdynamite/git-as

`git as` switches the git identity that you use for commits. For example, `git as frank` in one repository and `git as mantis` in another.

`git-as` is a git subcommand. Git finds `git-as` on your `PATH`, so you run it as `git as`.

## Install

Requires Go 1.24 or later.

```sh
make install
```

This installs the binary to `~/.local/bin`. Make sure that directory is on your `PATH`. To install somewhere else, set `INSTALL_DIR`, for example `make install INSTALL_DIR=/usr/local/bin`.

## Setup

Create one profile for each identity:

```sh
git as add frank --name "Frank Reynolds" --email "frank@example.com"
git as add mantis --name "Mantis Toboggan" --email "mantis@example.com" \
  --signingkey ~/.ssh/mantis.pub --sshcommand "ssh -i ~/.ssh/mantis"
```

Profiles are stored in your global git config (`~/.gitconfig`):

```ini
[profile "mantis"]
	name = Mantis Toboggan
	email = mantis@example.com
	signingkey = ~/.ssh/mantis.pub
	sshcommand = ssh -i ~/.ssh/mantis
```

`--signingkey` and `--sshcommand` are optional. Running `git as add` for an existing profile replaces all of its fields. Profile names can contain letters, digits, `.`, `_` and `-`. The names `add`, `remove`, `list` and `help` are reserved.

## Usage

```sh
git as                  # show the current identity and list profiles
git as frank            # use the "frank" profile in this repository
git as frank --global   # use the "frank" profile everywhere
git as list             # list profiles
git as remove mantis    # delete a profile
git as -h               # show help
```

Example output of `git as`:

```
current: Frank Reynolds <frank@example.com> (profile: frank)

profiles:
* frank   Frank Reynolds <frank@example.com>
  mantis  Mantis Toboggan <mantis@example.com>
```

Applying a profile sets `user.name` and `user.email`. It also sets `user.signingkey` and `core.sshCommand` if the profile defines them. If the profile does not define them, `git as` removes them at that scope, so a key from the previous profile does not stay active.

## Notes

- Use `git as -h` for help. Git handles `git as --help` itself and looks for a man page, which `git-as` does not install.
- Local settings override global ones, but global settings still apply where no local value exists. This is standard git behavior. For example, if your global config has a `user.signingkey` and you run `git as mantis` for a profile without one, the global key still applies in that repository.
- `git as` does not change `commit.gpgsign` or `gpg.format`. Configure commit signing yourself.

## Development

```sh
make build   # build ./git-as
make test    # run all tests
```

The integration tests use a temporary `HOME` and git repository. They never read or write your real git config.

## License

This project is released into the public domain under the [Unlicense](LICENSE). You can use it for any purpose, with no restrictions.
