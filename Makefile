# Enable soon (GO 1.12) to be deprecated
# flag indicating the go toolchain should
# be module aware as this package is a
# go module
export GO111MODULE=on
all: lint

.PHONY: all lint version

lint:
	go vet ./...
	go mod tidy

# target for tagging and publishing a new version of the SDK
# run like make version version=X.Y.Z
version:
	git tag v${version}
	git push origin v${version}
