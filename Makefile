.PHONY: build test vet clean

build:
	CGO_ENABLED=0 go build -o notionctl .

test:
	CGO_ENABLED=0 go test ./... -v

vet:
	go vet ./...

clean:
	rm -f notionctl
