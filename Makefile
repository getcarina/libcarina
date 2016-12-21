GOFILES = $(wildcard **/*.go)
GOFILES_NOVENDOR = $(shell go list ./... | grep -v /vendor/)

default: get-deps validate local

get-deps:
	go get github.com/Masterminds/glide
	glide install

validate:
	go fmt $(GOFILES_NOVENDOR)
	go vet $(GOFILES_NOVENDOR)
	go list ./... | grep -v /vendor/ | xargs -L1 golint --set_exit_status

local: $(GOFILES)
	go build .

test: local
	go test -v $(GOFILES_NOVENDOR)
