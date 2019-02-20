# utils-go

This module contains common functions and types that have utility across tozny golang based repositories.

## Build
To "build" this module, type

```
make lint
```

this will run the go psuedo-compiler and linter tool `vet`, along with the `mod` tool to handle golang source code dependency management.

## Development
Checkout branch to work on.
Write code on that branch.
Make a commit on that branch.
Push branch & commit.
From other repository that depends on this module, fetch the committed changes by running

```
go get github.com/tozny/utils-go@GITCOMMITSHA
```

Iterate on committed changes from within dependent repository.

## Publishing

Follow [semantic versioning](https://semver.org) when releasing new versions of this library.

Releasing involves tagging a commit in this repository, and pushing the tag. Tagging and releasing of new versions should only be done from the master branch after an approved Pull Request has been merged, or on the branch of an approved Pull Request.

To publish a new version, run

```
go tag vX.Y.Z
go push origin vX.Y.Z
```

To consume published updates from other repositories that depends on this module run

```
go get github.com/tozny/utils-go@vX.Y.Z
```

and the go `get` tool will fetch the published artifact and update that modules `go.mod` and`go.sum` files with the updated dependency. Currently the list of modules that depend on this module are

- [Search Service](https://github.com/tozny/hook-service)
- [Hook Service](https://github.com/tozny/e3dbSearchService)
