.PHONY: build clean installer

BINARY=tgtool.exe
VERSION=2.0.0

build:
	GOPROXY=https://goproxy.cn,direct go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/tgtool/

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o tgtool ./cmd/tgtool/

build-mac:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o tgtool ./cmd/tgtool/

installer: build
	makensis installer.nsi

clean:
	rm -f $(BINARY) tgtool-setup-*.exe

test:
	go test ./...
