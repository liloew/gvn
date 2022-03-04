GVN: Golang implementation VPN, aimed at distributed work environments.
---
# Build
```
# for cross-compile, you may need to install aarch64 compile tools:
# dnf install gcc-aarch64-linux-gnu gcc-c++-aarch64-linux-gnu # Fedora and others using RPM etc
# apt-get install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu # Ubuntu and others using DEB etc
make clean && make all
```

---
# Generate config file
```
export PATH_TO_GVN="..."
# server
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` init -c server.yaml -s
# client
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` init -c client.yaml
```

---
# Run
```
export PATH_TO_GVN="..."
# server
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` up -c server.yaml
# client
"${PATH_TO_TO_GVN}"/gvn-`uname`-`uname -m` up -c client.yaml
```

---
# Windows
```
Download the wintun.dll from the website and place it under %SYSTEM32% dir.
```

---
# Test
Has been tested on `Linux`, `macOS` and `Windows`.
