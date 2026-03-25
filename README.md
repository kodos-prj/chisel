# Chisel - Cross-Distribution Package Manager

Bring **Arch Linux packages to ANY Linux distribution**. Chisel runs Arch packages natively on Ubuntu, Fedora, Debian, and more with complete dependency isolation and zero host contamination.

## Purpose

Chisel is a **cross-distribution package manager** that solves a real problem: stable LTS distributions (Ubuntu 22.04, Debian 12) with outdated packages, but you need bleeding-edge development tools.

**Without Chisel:**
- Ubuntu 22.04 ships Python 3.10 (2021), but you need Python 3.12
- Option 1: Compile from source (time-consuming, error-prone)
- Option 2: Add PPAs (security risk, conflicts with system packages)
- Option 3: Upgrade entire OS (breaks stability)

**With Chisel:**
```bash
chisel sync
chisel install python  # Get Python 3.12 from Arch, isolated
```

Your system's Python 3.10 stays untouched, Arch's Python 3.12 works perfectly alongside it. No conflicts, no contamination.

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

### Core Features (v1.0)
- **Sync** Arch databases from mirrors (`chisel sync`)
- **Install** packages with complete dependency isolation (ALL deps from Arch)
- **Wrapper generation** for library path management
- **Remove** packages with orphan cleanup
- **Upgrade** packages safely
- **Query** installed packages and search repositories
- **Cleanup** old package versions from store
- **Cross-distribution** compatibility (Ubuntu, Fedora, Debian, Arch)
- **Fast & reliable** - compiled Go binary, minimal dependencies

### Future Features (v1.1+)
- **ARM64 architecture** support - v1.1
- **Package scripts** execution (install/remove hooks) - v1.1
- **Generation management** (system snapshots, rollback) - v2.0
- **Boot integration** (select versions at boot) - v2.0
- **Web UI** for package management - v2.0

### Architecture Highlights
- **Single binary** - only requires libalpm installed
- **7 main packages**: `config`, `registry`, `database` (sync), `alpm` (ALPM wrapper), `store` (package storage), `wrapper` (script generation), `symlink` (symlink management)
- **~5,000-7,000 lines** of Go code (estimated)
- **Filesystem agnostic** - works on ext4, xfs, btrfs, etc.
- **Concurrent operations** with goroutines
- **Complete isolation** - never mixes host and Arch libraries
- **Storage overhead** - 2-3x size (worth it for universal compatibility)

## Supported Distributions

### Tier 1 (Tested, Fully Supported)
- ✅ **Ubuntu 22.04 LTS** (glibc 2.35)
- ✅ **Ubuntu 24.04 LTS** (glibc 2.39)
- ✅ **Fedora 39** (glibc 2.38)
- ✅ **Fedora 40** (glibc 2.39)
- ✅ **Debian 12** (glibc 2.36)
- ✅ **Arch Linux** (glibc 2.39+) - Works natively

### Tier 2 (Compatible, Community Supported)
- ✅ **CentOS Stream 9** (glibc 2.34)
- ✅ **openSUSE Leap** (glibc 2.38)
- ✅ **Linux Mint** (Ubuntu-based)
- ✅ **Pop!_OS** (Ubuntu-based)

### Not Supported
- ❌ **Alpine Linux** (uses musl, not glibc)
- ❌ **Void Linux** (different package format)

**Requirement**: systemd-based glibc Linux distribution with kernel 3.10+

## Project Status

### Current State
**Phase**: Phase 0 complete (documentation updated to v4.0)  
**Version**: v4.0 (Cross-Distribution Architecture)  
**Status**: Ready to begin Phase 1 (Foundation & ALPM)

### v4.0 Changes (Cross-Distribution)
**Added in v4.0**:
- ✅ **Cross-distribution support** (Ubuntu, Fedora, Debian, Arch)
- ✅ **Complete dependency isolation** (ship ALL dependencies from Arch)
- ✅ **Wrapper script system** for library path management
- ✅ **Database sync** from Arch mirrors
- ✅ **Custom ALPM root** (`/kod/` instead of `/`)
- ✅ **Multi-distribution testing** strategy (Docker)
- ✅ **Extended timeline** to 7-9 weeks (from 6-8)

**Kept from v3.0**:
- ✅ Go implementation with go-alpm/v2
- ✅ Simplified architecture
- ✅ Central store + symlink model
- ✅ JSON registry for package tracking
- ✅ Deferred features (generation mgmt, package scripts to v1.1/v2.0)

**Rationale for Cross-Distribution**:
- **10x larger audience** - Ubuntu/Fedora/Debian users outnumber Arch users
- **Real problem solved** - Stable distro users need bleeding-edge tools
- **Unique value** - Only tool bringing Arch packages everywhere
- **Storage trade-off accepted** - 2-3x size for universal compatibility

### Implementation Progress
- ✅ Go module initialized with go-alpm/v2 dependency
- ✅ Project structure created (pkg/ for public packages)
- ✅ Config package COMPLETE (85% test coverage, JSON format)
- ✅ Registry package COMPLETE (75% test coverage)
- ✅ Basic CLI structure
- ⬜ Database sync system (Phase 1.2)
- ⬜ ALPM wrapper with /kod root (Phase 1.3)
- ⬜ Wrapper generation (Phase 3)
- ⬜ Package installation (Phase 4)
- ⬜ Multi-distro Docker testing (Phase 6)

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

## Timeline

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| **Phase 0**: Documentation | COMPLETE | Spec v4.0, cross-distro architecture docs |
| **Phase 1**: Foundation & ALPM | 1-1.5 weeks | Database sync, ALPM with /kod root |
| **Phase 2**: Storage & Extraction | 1 week | Package extraction, library discovery |
| **Phase 3**: Wrapper & Symlinks | 1 week | Wrapper generation, two-tier symlinks |
| **Phase 4**: Installation | 1.5-2 weeks | Full install with ALL dependencies |
| **Phase 5**: Removal & Queries | 1 week | Remove, orphans, queries, cleanup |
| **Phase 6**: Testing & Polish | 1 week | Docker multi-distro tests, bug fixes |
| **Phase 7**: Documentation | 3-5 days | User guide, FAQ, troubleshooting |
| **Total** | **7-9 weeks** | **Production-ready v1.0 (cross-distro)** |

**Comparison to v3.0**: Extended by 1-3 weeks for cross-distribution complexity (database sync, wrapper generation, multi-distro testing).

## Success Criteria

✅ **Functional** (v1.0 Cross-Distribution):
- Sync databases from Arch mirrors
- Install packages with complete dependency isolation (ALL deps from Arch)
- Generate wrapper scripts with correct LD_LIBRARY_PATH
- Works on Ubuntu 22.04+, Fedora 39+, Debian 12+, Arch Linux
- Remove packages with orphan cleanup
- Upgrade packages safely
- Query installed packages
- Search repositories
- Clean up old package versions
- Atomic operations (all-or-nothing)

✅ **Quality**:
- 80%+ unit test coverage
- Docker integration tests pass on Ubuntu, Fedora, Debian
- Zero critical bugs
- Clean, documented code
- Comprehensive user documentation
- Clear troubleshooting guide for each distribution

✅ **Performance**:
- Install 100 packages in < 10 minutes (slower due to full dependencies)
- Sync databases in < 30 seconds
- Remove packages in < 30 seconds
- Query response in < 1 second
- Acceptable storage overhead (2-3x)

✅ **Usability**:
- Intuitive CLI commands (similar to pacman)
- Helpful error messages with per-distribution troubleshooting
- Clear progress indicators
- Works identically across distributions
- Storage overhead warning displayed clearly

## Next Steps

### Immediate (Week 1-1.5 - Phase 1)
1. ✅ Review and update all documentation to v4.0
2. ⬜ Implement database sync system (download core.db, extra.db from mirrors)
3. ⬜ Initialize ALPM with /kod root and /kod/db paths
4. ⬜ Implement `chisel sync` command
5. ⬜ Implement `chisel search` and `chisel info` commands

### Short-term (Weeks 2-3 - Phases 2-3)
- Implement package extraction (zstd support)
- Build library path discovery
- Create wrapper script generation
- Implement two-tier symlink management (symlink → wrapper → binary)

### Medium-term (Weeks 4-5 - Phase 4)
- Complete package installation workflow
- Implement full dependency resolution (ALL deps including system libs)
- Handle errors and edge cases
- Test on Ubuntu, Fedora, Debian with Docker

### Long-term (Weeks 6-9 - Phases 5-7)
- Implement package removal
- Add query and search functionality
- Write comprehensive tests (80%+ coverage)
- Multi-distribution Docker testing
- Polish CLI, optimize performance
- Write user documentation and FAQ

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

This documentation is part of the chisel project. License TBD.

---

**Last Updated**: 2026-03-21  
**Status**: Phase 0 complete (documentation v4.0), ready for Phase 1 implementation  
**Version**: v4.0 (Cross-Distribution Architecture)  
**Timeline**: 7-9 weeks to v1.0  
**Target Distributions**: Ubuntu 22.04+, Fedora 39+, Debian 12+, Arch Linux
