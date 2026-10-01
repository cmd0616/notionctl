VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/radityajayantara/notionctl/cmd.version=$(VERSION)

.PHONY: build test vet clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o notionctl .

test:
	CGO_ENABLED=0 go test ./... -v

vet:
	go vet ./...

clean:
	rm -f notionctl
