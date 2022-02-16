.PHONY: all
all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/altgvn-linux-amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -o target/altgvn-linux-arm64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/altgvn-darwin-amd64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o target/altgvn-darwin-arm64 main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/altgvn-windows-amd64.exe main.go
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o target/altgvn-freebsd-amd64 main.go
macOS:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/altgvn-darwin-amd64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o target/altgvn-darwin-arm64 main.go
linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/altgvn-linux-amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -o target/altgvn-linux-arm64 main.go
windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/altgvn-windows-amd64.exe main.go
freebsd:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o target/altgvn-freebsd-amd64 main.go
clean:
	rm -rf target
format:
	go fmt ./...
