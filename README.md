# Chisel - Cross-Distribution Package Manager

Bring **Arch Linux packages to any Linux distribution**. Chisel runs Arch packages natively on Ubuntu, Fedora, Debian, and other systemd-based distributions with complete dependency isolation and zero host contamination.

## Purpose

Chisel solves a real problem: stable LTS distributions (Ubuntu 22.04, Debian 12) have outdated packages, but you need bleeding-edge development tools. Instead of wrestling with PPAs, backports, or containers, Chisel lets you install Arch packages directly with all their dependencies isolated from your host system.

## Key Innovation

Chisel brings Arch packages to any distribution by:
1. **Complete dependency isolation** - ALL dependencies from Arch (glibc, gcc-libs, everything)
2. **Wrapper scripts** - Set `LD_LIBRARY_PATH` dynamically so binaries load Arch libraries, not host libraries
3. **Custom ALPM root** - Uses `/kod/` instead of `/`, works independently of host package manager
4. **Database sync** - Downloads Arch package databases directly from mirrors
5. **Package groups** - Install entire collections like GNOME, KDE, or development tools in one command

**Result**: Arch binaries work identically on Ubuntu, Fedora, Debian, and other distributions.

## Quick Start

### System-Level Installation (requires sudo)

```bash
# Build the project (pure Go, no system dependencies)
go build -o chisel ./cmd/chisel

# Sync Arch databases
sudo ./chisel sync

# Install a package (with ALL dependencies from Arch)
sudo ./chisel install vim

# Install an entire package group
sudo ./chisel install gnome

# Search for packages
sudo ./chisel search --groups          # List all 100+ available groups
sudo ./chisel search --group pro-audio # Show packages in pro-audio group
sudo ./chisel search vim               # Search for specific package

# Run tests
go test ./...
```

### User-Level Package Management (no sudo!)

Want to install packages **without sudo**? Chisel supports user-level package management!

```bash
# One-time setup (creates ~/.local/share/chisel and ~/.config/chisel)
./chisel-user-init.sh

# Reload your shell
source ~/.bashrc

# Now install packages without sudo!
chisel-user sync
chisel-user install vim
chisel-user install nano
chisel-user install gnome          # Install entire group
chisel-user search --groups        # Discover groups
chisel-user list
chisel-user upgrade
chisel-user cleanup
```

**Key Benefits:**
- ✅ No sudo required
- ✅ Packages isolated in `~/.local/share/chisel/`
- ✅ Executables in `~/.local/bin/`
- ✅ Per-user isolation
- ✅ Follows XDG directory standards
- ✅ Easy setup and cleanup

See [USER-GUIDE.md](USER-GUIDE.md) for comprehensive documentation.

## Comparison: System vs User-Level

| Feature | System (sudo) | User-Level |
|---------|--------------|-----------|
| Installation | `sudo chisel install vim` | `chisel-user install vim` |
| Location | `/kod/` (global) | `~/.local/share/chisel/` (per-user) |
| Permissions | Root required | No sudo needed |
| Isolation | Shared across users | Per-user |
| Usage | Server/CI environments | Development, personal use |
| Setup | Direct use | `chisel-user-init.sh` once |

## Features

### Package Management
- **Sync** Arch databases from mirrors (`chisel sync`)
- **Install** packages with complete dependency isolation (ALL deps from Arch)
- **Install package groups** (`chisel install gnome` installs 50+ packages)
- **Search packages** with multiple options:
  - By name: `chisel search vim`
  - By group: `chisel search --group kde-applications`
  - List all groups: `chisel search --groups`
- **Query** installed packages and search repositories
- **Remove** packages with orphan cleanup
- **Upgrade** packages safely
- **Cleanup** old package versions from store

### Package Sources
- **Official Arch repositories** - core, extra (community removed in v0.2.0+)
- **AUR (Arch User Repository)** - building packages from source with full dependency resolution
- **Automatic source detection** - distinguishes between official and AUR packages
- **Source constraints** - `--source=official` or `--source=aur` to limit search scope

### Advanced Dependency Resolution
- **Version constraint parsing** - handles dependencies like `linux-api-headers>=4.10`
- **Virtual package resolution** - resolves packages providing virtual names (e.g., `libncursesw.so`)
- **Circular dependency handling** - gracefully handles legitimate cycles in Arch databases
- **Mixed resolver** - seamlessly resolves dependencies across official and AUR packages

### Chroot & Container Support
- **Symlink-prefix stripping** - adjust paths for containers and chroots (`--symlink-prefix /tmp/demo`)
- **Portable packages** - run packages in different mount points and environments
- **Cross-environment compatibility** - use same packages in different contexts

### System Architecture
- **Single binary** - pure Go, no external dependencies (NO libalpm required)
- **7 main packages**: `config`, `registry`, `database` (sync), `alpm` (package parsing), `store` (package storage), `wrapper` (script generation), `symlink` (symlink management), `build` (AUR builder)
- **~10,000+ lines** of Go code with comprehensive tests
- **Filesystem agnostic** - works on ext4, xfs, btrfs, etc.
- **Concurrent operations** with goroutines
- **Complete isolation** - never mixes host and Arch libraries
- **Storage overhead** - 2-3x size (worth it for universal compatibility)

## What is Chisel?

A **cross-distribution package manager** that:
- Brings **Arch Linux packages** to Ubuntu, Fedora, Debian, and other systemd-based distributions
- Uses **complete dependency isolation** - ALL dependencies from Arch, not host system
- Creates **wrapper scripts** that set `LD_LIBRARY_PATH` for library isolation
- Uses a **central package store** at `/kod/store/<package>/<version>/`
- Stores **wrapper scripts** at `/kod/wrappers/`
- Creates **two-tier symlinks**: `/usr/bin/vim` → `/kod/wrappers/vim` → `/kod/store/vim/9.0/usr/bin/vim`
- Tracks packages in a **JSON registry** at `/kod/registry.json`
- Syncs **Arch databases** from mirrors to `/kod/db/`
- Works **independently** of host package manager (apt, dnf, pacman)
- Written in **pure Go** for performance, safety, and single-binary deployment

## Technology Stack

### Core
- **Language**: Go 1.25+ (no external C dependencies)
- **ALPM Implementation**: Pure Go (no libalpm required)
- **CLI Framework**: Native Go `flag` package (not Cobra)
- **Package Format**: Native Arch packages (.pkg.tar.zst)
- **Package Source**: Arch Linux mirrors (core, extra repositories)
- **Filesystem**: Any POSIX (ext4, xfs, btrfs, etc.)
- **Distribution Requirements**: systemd-based glibc Linux, kernel 3.10+

### CLI
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
- **Coverage**: 249+ tests, targeting 80%+
- **Integration Tests**: Real Arch databases cached for testing
- **Multi-Distribution**: Tested on Ubuntu, Fedora, Debian

## Version History

### v0.3.0 (Current) - April 2026
- ✨ **Package Groups**: Install entire collections with one command
  - `chisel install gnome` (installs 50+ packages)
  - `chisel search --groups` (lists 100+ groups)
  - `chisel search --group <name>` (find packages in group)
- 🐛 **Dependency Resolution Improvements**:
  - Parse version constraints in dependencies
  - Resolve virtual packages from Provides field
  - Gracefully handle circular dependencies
- 📦 **AUR Integration**: Build and install packages from Arch User Repository
- 🔗 **Symlink Prefix Stripping**: For container and chroot support

### v0.2.0 - April 2026
- ✨ Symlink-prefix stripping for chroot environments
- 📚 Generation-based registry management specification

### v0.1.0 - April 2026
- 🎯 Core package installation functionality
- 📦 Official Arch repository support
- 🔄 Mixed official/AUR dependency resolution
- 👤 User-level package management

## Installation & Building

### Prerequisites
- **Any Linux distribution** (Ubuntu 22.04+, Fedora 39+, Debian 12+, Arch, etc.)
- Go 1.25 or higher
- Root access (for system-level installation testing)

### Building
```bash
# Build the project (pure Go, no system dependencies)
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
│   └── chisel/              # Main CLI entry point
├── pkg/                     # Public reusable packages
│   ├── alpm/                # Package parsing (pure Go ALPM)
│   ├── build/               # AUR builder and mixed resolver
│   ├── config/              # Configuration (JSON)
│   ├── database/            # Database sync from mirrors
│   ├── registry/            # Package registry tracking
│   ├── store/               # Package store management
│   ├── symlink/             # Symlink operations
│   ├── download/            # Package downloads
│   ├── extract/             # Package extraction
│   ├── aur/                 # AUR integration
│   └── wrapper/             # Wrapper script generation
├── internal/
│   └── cli/                 # CLI commands
├── integration/             # Integration tests
├── go.mod
└── README.md
```

## Documentation

See the following documentation files for detailed information:

- **[USER-GUIDE.md](USER-GUIDE.md)** - Complete user guide for both system and user-level usage
- **[CHANGELOG.md](CHANGELOG.md)** - Detailed version history and changes
- **[DEVELOPER-GUIDE.md](DEVELOPER-GUIDE.md)** - Development setup and architecture details
- **[docs/CONFIGURATION.md](docs/CONFIGURATION.md)** - Configuration file format and options
- **[docs/AUR_INTEGRATION.md](docs/AUR_INTEGRATION.md)** - AUR support details

## Resources

### External Documentation
- [Arch Linux Wiki - Pacman](https://wiki.archlinux.org/title/Pacman)
- [Arch Linux Mirrors](https://archlinux.org/mirrors/) - Mirror list for database sync
- [ALPM Package Format](https://wiki.archlinux.org/title/Creating_packages)
- [Go Best Practices](https://go.dev/doc/effective_go)

### Distribution Resources
- [Ubuntu Packages](https://packages.ubuntu.com/) - Check Ubuntu package versions
- [Fedora Packages](https://packages.fedoraproject.org/) - Check Fedora package versions
- [Debian Packages](https://packages.debian.org/) - Check Debian package versions

## Contributing

When working on this project:

1. **Write tests** - Aim for 80%+ coverage
2. **Test on multiple distros** - Test on Ubuntu, Fedora, and Debian
3. **Document as you go** - Add comments and update docs
4. **Keep it simple** - Focus on core functionality
5. **Update CHANGELOG** - Document all changes

## Use Cases

### Desktop Environment Installation
```bash
# Install complete GNOME desktop
chisel install gnome

# Or KDE Plasma
chisel install kde-applications

# Or development tools
chisel install base-devel
```

### Development Tools
```bash
# Install cutting-edge development packages
chisel install gcc git vim neovim

# Install pro-audio tools
chisel install pro-audio

# Get the latest Node.js from AUR
chisel install nodejs  # Automatically uses official or AUR
```

### Container/Chroot Environments
```bash
# Install packages with adjusted paths for containers
chisel install --symlink-prefix /opt/app vim

# Create portable package environments
chisel install base-devel --symlink-prefix /mnt/build
```

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
