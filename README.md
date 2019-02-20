# utils-go

Purpose....

## Build
To "build" this package/run the go compiler/linter type

```
make lint
```

## Development
Checkout out branch
Write code on that branch
Make a commit on that branch
Push branch & commit
From other repository that depends on this package

```
go get github.com/tozny/utils-go@GITCOMMITSHA
```

Iterate on committed changes

## Publishing

Use [semantic versioning](https://semver.org)

```
go tag vX.Y.Z
go push origin vX.Y.Z
```

From other repository that depends on this package

```
go get github.com/tozny/utils-go@vX.Y.Z
```
