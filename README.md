# Repo guidelines

## Pre-commit configuration
1. Install `golangci-lint` and `gofumpt` in your system.
2. Install pre-commit: `pip install pre-commit` or through your package manager
3. Install hooks: `pre-commit install --hook-type pre-commit --hook-type commit-msg`
4. Update the repository hooks: `pre-commit autoupdate`

## Release process
The release process is automated using **goreleaser** which works in *snapshot* mode on the `develop` branch in order to test the release process.    
Meanwhile, the release process is triggered by assigning a **tag** in a commit where the tag name is in the format `vX.Y.Z`, following the semantic versioning. So you can create a release in any branch, but it is recommended to create a release from the `main` branch or to merge into `main` right after the release.

## Commit message
The commit messages must follow the [Conventional Commits](https://www.conventionalcommits.org) specification, which is a lightweight convention on top of commit messages. It provides an easy set of rules for creating an explicit commit history; which makes it easier to write automated tools on top of.  
Only the following commit types are included in the changelog: `feat`, `feat!`, `fix`, `fix!`, `BREAKING CHANGE`.