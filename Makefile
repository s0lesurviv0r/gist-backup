.PHONY: build

build:
	go build -o build/gist-backup main.go

test:
	go test -v ./...

clean:
	rm -rf build

fmt:
	gofmt -w **/*.go
