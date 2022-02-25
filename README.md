---
# Init
```
mkdir gvn && cd gvn
go mod init github.com/liloew/gvn
git init . && git add . && git commit -m "Init"
# Add cobra.yaml
cobra init --config cobra.yaml
git add . && git commit -m "cobra init"
cobra add init --config cobra.yaml
cobra add up --config cobra.yaml
cobra add down --config cobra.yaml
```

---
# Build
```
# for cross-compile:
# dnf install gcc-aarch64-linux-gnu gcc-c++-aarch64-linux-gnu # Fedora and others using RPM etc
# apt-get install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu # Ubuntu and others using DEB etc
make clean && make all
```

---
# Generate config file
```
# server
export PATH_TO_GVN="..."
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` init -c server.yaml -s
# client
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` init -c client.yaml
```

---
# Run
```
# server
export PATH_TO_GVN="..."
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` up -c server.yaml
# client
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` up -c client.yaml
```

---
# Windows
```
Download the wintun.dll from the website and place it under %SYSTEM32% dir.
```
