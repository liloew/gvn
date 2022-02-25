#!/bin/sh
# Copyright © 2022 liluo <luolee.me@gmail.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
.PHONY: all
all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/gvn-Linux-x86_64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -o target/gvn-Linux-aarch64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/gvn-Darwin-x86_64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o target/gvn-Darwin-aarch64 main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/gvn-Windows-x86_64.exe main.go
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o target/gvn-Freebsd-x86_64 main.go
macOS:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/gvn-Darwin-x86_64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o target/gvn-Darwin-aarch64 main.go
linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/gvn-Linux-x86_64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ go build -o target/gvn-Linux-x86_64 main.go
windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/gvn-Windows-x86_64.exe main.go
freebsd:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o target/gvn-FreeBSD-amd64 main.go
clean:
	rm -rf target
format:
	go fmt ./...
