---
title: "Installation"
description: "All ways to install gotk: go install, pre-built binaries, and building from source."
order: 2
---

# Installation

## go install

Requires Go 1.22 or later. The binary is placed in `$GOPATH/bin` (usually `~/go/bin`).

```bash
go install github.com/omurilo/gotk@latest
```

Make sure `~/go/bin` is in your `PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

## Pre-built binaries

Download the latest release from the [Releases page](https://github.com/omurilo/gotk/releases).

| OS | Architecture | File |
|---|---|---|
| macOS | Apple Silicon (M1/M2/M3) | `gotk_darwin_arm64.tar.gz` |
| macOS | Intel | `gotk_darwin_amd64.tar.gz` |
| Linux | x86-64 | `gotk_linux_amd64.tar.gz` |
| Linux | ARM64 | `gotk_linux_arm64.tar.gz` |
| Windows | x86-64 | `gotk_windows_amd64.zip` |

### Verify the checksum

```bash
# Download the checksum file alongside the archive
curl -LO https://github.com/omurilo/gotk/releases/latest/download/checksums.txt
sha256sum --check checksums.txt --ignore-missing
```

### Extract and install (Unix)

```bash
tar -xzf gotk_linux_amd64.tar.gz
sudo mv gotk /usr/local/bin/
gotk --version
```

### Windows

Extract the `.zip`, place `gotk.exe` in a directory on your `PATH`, and verify:

```powershell
gotk.exe --version
```

## Build from source

```bash
git clone https://github.com/omurilo/gotk
cd gotk
go build -o gotk .
sudo mv gotk /usr/local/bin/
```

To embed version info:

```bash
VERSION=$(git describe --tags --always)
go build -ldflags "-s -w -X main.version=$VERSION" -o gotk .
```

## Uninstall

```bash
sudo rm /usr/local/bin/gotk
rm -rf ~/.config/gotk
rm -rf ~/.local/share/gotk

# Remove the agent hook
gotk init --uninstall
# or for a specific agent:
gotk init --agent cursor --uninstall
```
