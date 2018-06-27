.PHONY: build clean test help default lint

PROJECT=github.com/praekeltfoundation/vault-plugin-auth-mesos

BIN_NAME=vault-plugin-auth-mesos

VERSION := $(shell grep "const Version " version/version.go | sed -E 's/.*"(.+)"$$/\1/')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)

default: test

help:
	@echo 'Management commands for vault-plugin-auth-mesos:'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project.'
	@echo '    make get-deps        Run dep ensure, mostly used for ci.'
	@echo '    make test            Run tests on a compiled project.'
	@echo '    make lint            Run gometalinter.'
	@echo '    make clean           Clean the directory tree.'
	@echo

build:
	@echo "building ${BIN_NAME} ${VERSION} ${GIT_COMMIT}${GIT_DIRTY}"
	@echo "GOPATH=${GOPATH}"
	go build -ldflags "-X ${PROJECT}/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X ${PROJECT}/version.VersionPrerelease=DEV" -o bin/${BIN_NAME} cmd/${BIN_NAME}/main.go

get-deps:
	dep ensure

clean:
	@test ! -e bin/${BIN_NAME} || rm bin/${BIN_NAME}

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic $(shell go list ./... | fgrep -v '/cmd/')

lint:
	gometalinter --vendor --tests --deadline=120s ./...
