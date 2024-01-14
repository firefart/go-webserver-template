PROG=template

.DEFAULT_GOAL := build

.PHONY: all
all: update lint build

.PHONY: docker-update
docker-update:
	docker pull golang:latest
	docker pull scratch:latest
	docker build --tag ${PROG}:dev .

.PHONY: lint
lint:
	"$$(go env GOPATH)/bin/golangci-lint" run ./...
	go mod tidy

.PHONY: lint-update
lint-update:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	$$(go env GOPATH)/bin/golangci-lint --version

.PHONY: update
update:
	go get -u
	go mod tidy -v

.PHONY: build
build:
	go fmt ./...
	go vet ./...
	go build -o ${PROG}

.PHONY: test
test:
	go test -race -cover ./...

.PHONY: run
run: build
	 ./${PROG} -debug -c config.json
