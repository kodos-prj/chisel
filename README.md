# Chisel - Cross-Distribution Package Manager

Bring **Arch Linux packages to ANY Linux distribution**. Chisel runs Arch packages natively on Ubuntu, Fedora, Debian, and more with complete dependency isolation and zero host contamination.

## Purpose

Chisel is a **cross-distribution package manager** that solves a real problem: stable LTS distributions (Ubuntu 22.04, Debian 12) with outdated packages, but you need bleeding-edge development tools.

## Key Innovation

Chisel brings Arch packages to ANY distribution by:
1. **Complete dependency isolation** - ALL dependencies from Arch (glibc, gcc-libs, everything)
2. **Wrapper scripts** - Set `LD_LIBRARY_PATH` dynamically so binaries load Arch libraries, not host libraries
3. **Custom ALPM root** - Uses `/kod/` instead of `/`, works independently of host package manager
4. **Database sync** - Downloads Arch package databases directly from mirrors

**Result**: Arch binaries work identically on Ubuntu, Fedora, Debian, etc.

## Quick Start

### For Building & Running (on ANY distribution!)
```bash
# Install libalpm (required)
# Ubuntu/Debian:
sudo apt-get install libalpm-dev

# Fedora:
sudo dnf install libalpm-devel

# Arch (already installed):
# pacman -S pacman

# Build the project
go build -o chisel ./cmd/chisel

# Sync Arch databases
sudo ./chisel sync

# Install a package (with ALL dependencies from Arch)
sudo ./chisel install vim

# Run tests
go test ./...
```

## Key Concepts

### What is Chisel?
A **cross-distribution package manager** that:
- Brings **Arch Linux packages** to Ubuntu, Fedora, Debian, and other distributions
- Uses **complete dependency isolation** - ALL dependencies from Arch, not host system
- Creates **wrapper scripts** that set `LD_LIBRARY_PATH` for library isolation
- Uses a **central package store** at `/kod/store/<package>/<version>/`
- Stores **wrapper scripts** at `/kod/wrappers/`
- Creates **two-tier symlinks**: `/usr/bin/vim` → `/kod/wrappers/vim` → `/kod/store/vim/9.0/usr/bin/vim`
- Tracks packages in a **JSON registry** at `/kod/registry.json`
- Syncs **Arch databases** from mirrors to `/kod/db/`
- Works **independently** of host package manager (apt, dnf, pacman)
- Written in **Go** for performance, safety, and single-binary deployment

### Core Features
- **Sync** Arch databases from mirrors (`chisel sync`)
- **Install** packages with complete dependency isolation (ALL deps from Arch)
- **Wrapper generation** for library path management
- **Remove** packages with orphan cleanup
- **Upgrade** packages safely
- **Query** installed packages and search repositories
- **Cleanup** old package versions from store
- **Cross-distribution** compatibility (Ubuntu, Fedora, Debian, Arch)
- **Fast & reliable** - compiled Go binary, minimal dependencies

### Architecture Highlights
- **Single binary** - only requires libalpm installed
- **7 main packages**: `config`, `registry`, `database` (sync), `alpm` (ALPM wrapper), `store` (package storage), `wrapper` (script generation), `symlink` (symlink management)
- **~5,000-7,000 lines** of Go code (estimated)
- **Filesystem agnostic** - works on ext4, xfs, btrfs, etc.
- **Concurrent operations** with goroutines
- **Complete isolation** - never mixes host and Arch libraries
- **Storage overhead** - 2-3x size (worth it for universal compatibility)

## Technology Stack

### Core
- **Language**: Go 1.21+
- **ALPM Bindings**: github.com/Jguer/go-alpm/v2 (v2.3.1)
- **Package Format**: Native Arch packages (.pkg.tar.zst)
- **Package Source**: Arch Linux mirrors (core, extra, community repos)
- **Filesystem**: Any POSIX (ext4, xfs, btrfs, etc.)
- **Distribution Requirements**: systemd-based glibc Linux, kernel 3.10+

### CLI
- **Framework**: Cobra (for complex subcommands)
- **Output**: Color-coded text, progress indicators
- **Config**: JSON format (`/etc/chisel/config.json`)

### Data Storage
- **Registry**: JSON file at `/kod/registry.json`
- **Package Store**: Directory structure at `/kod/store/`
- **Databases**: Arch databases at `/kod/db/` (synced from mirrors)
- **Wrappers**: Shell scripts at `/kod/wrappers/`
- **Configuration**: `/etc/chisel/config.json` (JSON, not YAML)

### Testing
- **Unit Tests**: Go's built-in testing package
- **Coverage**: Target 80%+
- **Integration**: Docker containers (Ubuntu 22.04, Fedora 40, Debian 12)
- **Multi-Distribution**: Automated testing across all Tier 1 distributions

## Resources

### Go Dependencies
- [go-alpm/v2](https://github.com/Jguer/go-alpm) - ALPM bindings for Go (v2.3.1, Nov 2025)
- [go-alpm docs](https://pkg.go.dev/github.com/Jguer/go-alpm/v2) - Package documentation and examples
- [cobra](https://github.com/spf13/cobra) - CLI framework

### External Documentation
- [Arch Linux Wiki - Pacman](https://wiki.archlinux.org/title/Pacman)
- [Arch Linux Mirrors](https://archlinux.org/mirrors/) - Mirror list for database sync
- [libalpm Documentation](https://archlinux.org/pacman/libalpm.3.html)
- [ALPM Package Format](https://wiki.archlinux.org/title/Creating_packages)
- [Go Best Practices](https://go.dev/doc/effective_go)

### Distribution Resources
- [Ubuntu Packages](https://packages.ubuntu.com/) - Check Ubuntu package versions
- [Fedora Packages](https://packages.fedoraproject.org/) - Check Fedora package versions
- [Debian Packages](https://packages.debian.org/) - Check Debian package versions

## Development

### Prerequisites
- **Any Linux distribution** (Ubuntu 22.04+, Fedora 39+, Debian 12+, Arch, etc.)
- Go 1.21 or higher
- libalpm installed:
  - Ubuntu/Debian: `sudo apt-get install libalpm-dev`
  - Fedora: `sudo dnf install libalpm-devel`
  - Arch: `sudo pacman -S pacman` (already installed)
- Docker or Podman (for multi-distribution testing)
- Root access (for testing actual package operations)

### Building
```bash
# Install libalpm (one-time)
# Ubuntu/Debian: sudo apt-get install libalpm-dev
# Fedora: sudo dnf install libalpm-devel
# Arch: sudo pacman -S pacman

# Build the project
go build -o chisel ./cmd/chisel

# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Project Structure
```
chisel/
├── cmd/
│   └── chisel/          # Main CLI entry point
├── pkg/                  # Public reusable packages
│   ├── config/           # Configuration (JSON)
│   ├── registry/         # Package registry
│   ├── alpm/             # ALPM wrapper (/kod root)
│   ├── database/         # Database sync from mirrors
│   ├── store/            # Package store management
│   ├── wrapper/          # Wrapper script generation
│   ├── symlink/          # Symlink operations
│   └── install/          # Installation orchestrator
├── internal/
│   └── cli/              # CLI commands
├── docs/                 # Documentation files
├── tests/                # Integration tests
│   └── docker/           # Multi-distro Docker tests
├── go.mod
└── README.md
```

### Testing on Multiple Distributions
Use Docker to test on different distributions:
```bash
# Example: Test on Ubuntu 22.04
docker run -it chisel-test-ubuntu bash
./run-tests.sh
```
Build Docker test images for your target distributions in `tests/docker/`.


### Contributing

When working on this project:

1. **Write tests** - Aim for 80%+ coverage
2. **Test on multiple distros** - Use Docker for Ubuntu, Fedora, Debian testing
3. **Document as you go** - Add comments and update docs
4. **Keep it simple** - Focus on v1.0 scope, defer features to v1.1/v2.0

## Documentation

See `docs/` directory for detailed documentation:
- **CONFIGURATION.md** - Configuration file format and options

## License

MIT License

Copyright (c) 2026 Chisel Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

