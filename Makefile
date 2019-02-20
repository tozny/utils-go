# Enable soon (GO 1.12) to be deprecated
# flag indicating the go toolchain should
# be module aware as this package is a
# go module
export GO111MODULE=on
all : lint

.PHONY : all lint

lint :
	go vet ./...
	go mod tidy
