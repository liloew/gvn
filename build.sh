rm -rf target
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/altgvn-linux-amd64 main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/altgvn-windows-amd64.exe main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/altgvn-darwin-amd64 main.go
