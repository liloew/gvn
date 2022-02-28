#!/bin/sh
# Copyright © 2022 lilo <luolee.me@gmail.com>
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

rm -rf target
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/gvn-linux-amd64 main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/gvn-windows-amd64.exe main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/gvn-darwin-amd64 main.go
