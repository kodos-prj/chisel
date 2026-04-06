# AUR Integration Guide

This document describes the Arch User Repository (AUR) integration in Chisel, enabling cross-distribution package management with AUR support.

## Overview

Chisel now supports installing packages from the Arch User Repository (AUR) in addition to official Arch repositories. This allows users to access a much larger ecosystem of packages while maintaining full recursive dependency resolution across both official and AUR sources.

### Key Features

- **Automatic AUR Fallback**: Official repositories are checked first; AUR is used as a fallback
- **Mixed Dependency Resolution**: Automatic resolution of dependencies that span both official and AUR sources
- **Build Management**: Integrated build system for compiling AUR packages with makepkg
- **Version Tracking**: Built-in tracking of package sources (official vs. AUR) and version history
- **Cleanup Integration**: Automatic cleanup of old build artifacts and logs
- **Transparent Operation**: Works seamlessly with existing Chisel commands (install, search, info, upgrade, cleanup)

## Architecture

### Components

The AUR integration consists of several integrated components:

#### 1. AUR Module (`pkg/aur/`)

The AUR module provides the foundation for AUR operations:

- **RPC Client** (`rpc.go`): Communicates with the official AUR RPC interface
  - Caches results with 24-hour TTL to reduce API calls
  - Implements rate limiting (4000 requests per day maximum)
  - Handles network errors and timeouts gracefully

- **Git Handler** (`git.go`): Manages PKGBUILD repository operations
  - Clones AUR repositories to the build cache
  - Verifies repository integrity
  - Handles authentication transparently

- **PKGBUILD Parser** (`pkgbuild.go`): Extracts package metadata
  - Parses bash array syntax in PKGBUILDs
  - Extracts dependencies, build dependencies, and version information
  - Handles complex PKGBUILDs with arrays and conditional logic

#### 2. Build System (`pkg/build/`)

The build system manages AUR package compilation:

- **BuildManager**: Orchestrates the build process
  - Executes `makepkg` with safe defaults (-s, -r, -C flags)
  - Maintains persistent build cache at `/kod/build-cache/`
  - Saves build logs to `/kod/build-logs/`
  - Provides age-based cleanup for old builds (default 7 days)

- **Build Artifact Management**:
  - Extracts built packages (.pkg.tar.zst files)
  - Integrates built packages into Chisel's package store
  - Tracks artifact paths and build results

#### 3. Mixed Dependency Resolver (`pkg/build/resolver.go`)

The resolver handles complex dependency chains:

- **Recursive Resolution**: Follows dependency chains through multiple levels
- **Mixed Sources**: Resolves dependencies from both official and AUR sources
- **Cycle Detection**: Prevents infinite loops in circular dependencies
- **Source Priority**: Always prefers official repositories over AUR
- **Validation**: Ensures all dependencies can be satisfied

#### 4. Registry Enhancement (`pkg/registry/`)

The registry now tracks package sources and version history:

- **Source Field**: Records whether a package came from official repos or AUR
- **Repository Field**: Tracks which AUR repository or official repo provides the package
- **Update Date Field**: Records when packages were installed/updated
- **Version History**: Maintains history of installed versions

#### 5. CLI Integration (`internal/cli/`)

All user-facing commands now support AUR:

- **SearchCommand**: Searches official repos first, then AUR if not found
- **InfoCommand**: Displays package info from either source
- **InstallCommand**: Uses MixedResolver for unified installation
- **UpgradeCommand**: Detects AUR packages and checks for updates via RPC
- **CleanupCommand**: Integrates AUR build cache and logs cleanup

## Usage

### Installing AUR Packages

```bash
# Install an AUR package (automatically built and integrated)
chisel install yay

# Install multiple packages (mix of official and AUR)
chisel install firefox chromium-browser

# Install with dependency resolution across sources
chisel install some-aur-package
# This will automatically resolve and install all dependencies,
# from either official repos or AUR as needed
```

### Searching for Packages

```bash
# Search for packages (searches official repos first, then AUR)
chisel search "python"

# Get information about a package
chisel info some-aur-package
```

### Upgrading Packages

```bash
# Upgrade all packages (including AUR)
chisel upgrade

# Upgrade specific package
chisel upgrade yay
```

### Cleaning Up

```bash
# Standard cleanup (removes old package versions)
chisel cleanup

# Cleanup with AUR build cache (removes old builds and logs)
chisel cleanup --aur

# Verbose cleanup to see details
chisel cleanup --aur --verbose

# Force cleanup without confirmation
chisel cleanup --aur --force
```

## Directory Structure

### Build Cache and Logs

The AUR build system uses the following directories:

```
/kod/
├── build-cache/          # Persistent cache for build directories
│   └── [package]-[version]-[timestamp]/  # Individual build directories
└── build-logs/           # Build log files
    └── [package]-[version].log           # Build output logs
```

### Package Store

Built AUR packages are stored identically to official packages:

```
/path/to/store/
└── [package-name]/
    ├── [version]/        # Contains extracted package files
    │   └── bin/
    │       └── executable
    └── [another-version]/
```

## Configuration

### Build Options

Chisel uses the following `makepkg` options for AUR packages:

- `-s`: Install missing dependencies using pacman
- `-r`: Remove build files after successful build
- `-C`: Skip integrity checks (user assumes responsibility)

### Cleanup Options

Default cleanup settings (configurable via command-line):

- Build cache retention: 7 days (older builds are removed)
- Build logs retention: 7 days (older logs are removed)
- Package version retention: configurable via `keep_versions` in config

## Workflow Examples

### Example 1: Simple AUR Package Installation

```bash
# Install yay (an AUR helper)
$ chisel install yay
  Checking official repositories...
  Package not found in official repos
  Checking AUR...
  ✓ Found yay in AUR
  ✓ Resolving dependencies (0 dependencies needed)
  ✓ Cloning PKGBUILD from AUR
  ✓ Building package with makepkg...
  ✓ Built yay successfully
  ✓ Installed yay 12.1.0 from AUR
```

### Example 2: Mixed Dependency Resolution

```bash
# Install a package with dependencies spanning both sources
$ chisel install some-aur-tool
  Checking official repositories...
  Package not found in official repos
  Checking AUR...
  ✓ Found some-aur-tool in AUR
  
  Resolving dependencies...
  ├─ dep1: found in official repos
  ├─ dep2: found in AUR (will be built)
  ├─ dep2 requires dep3 (official repos)
  └─ All dependencies resolved ✓
  
  Installing dependencies...
  ✓ Installed dep3 1.0.0 from official repos
  ✓ Built and installed dep2 2.1.0 from AUR
  ✓ Installed dep1 3.0.0 from official repos
  ✓ Built and installed some-aur-tool 1.2.0 from AUR
```

### Example 3: Cleanup with AUR

```bash
# Run cleanup to remove old builds
$ chisel cleanup --aur --verbose
  All packages are at their current versions. No cleanup needed.
  
  Cleaning AUR build cache and logs...
    ✓ Removed 3 old build directory(ies)
    ✓ Removed 5 old log file(s)
    ✓ Freed 256.50 MB from AUR cache
  
  ✓ Cleanup Summary:
    AUR cleanup:
      Build directories: 3
      Log files:         5
      Space freed:       256.50 MB
```

## Version Tracking

### Package Source Identification

Installed packages track their source in the registry:

```json
{
  "packages": {
    "yay": {
      "version": "12.1.0",
      "source": "aur",
      "repository": "yay",
      "update_date": "2024-04-05T12:30:45Z",
      "executables": ["yay"]
    },
    "firefox": {
      "version": "124.0",
      "source": "official",
      "repository": "extra",
      "update_date": "2024-04-04T10:15:30Z",
      "executables": ["firefox"]
    }
  }
}
```

### Version History

```bash
# View installed packages filtered by source
$ chisel list --aur          # Show only AUR packages
$ chisel list --official     # Show only official packages
$ chisel list                # Show all packages
```

## Security Considerations

### User Responsibility

Users installing AUR packages assume responsibility for:

1. **PKGBUILD Review**: Chisel does not sandbox package builds. Users should review PKGBUILDs before installation:
   ```bash
   # Inspect the PKGBUILD before installation
   chisel inspect-pkgbuild yay
   ```

2. **Build Dependencies**: Ensure `base-devel` is installed:
   ```bash
   pacman -S base-devel
   ```

3. **Dependency Chain Review**: For complex dependencies, review what will be installed:
   ```bash
   chisel info --deps some-aur-package
   ```

### Build Safety

- Builds run with standard user privileges (not sandboxed)
- Build artifacts are verified after compilation
- Failed builds don't pollute the package store

## Troubleshooting

### Common Issues

#### 1. Build Failures

```bash
# Check the build log for the specific error
$ cat /kod/build-logs/yay-12.1.0.log

# Common causes:
# - Missing build dependencies (install base-devel)
# - Incompatible PKGBUILD (report to AUR maintainer)
# - Network issues during build (retry)
```

#### 2. Dependency Resolution Failures

```bash
# If a dependency can't be found in either source:
$ chisel search missing-dep  # Verify package exists
$ chisel info missing-dep    # Check for typos

# If circular dependencies are detected:
chisel install package-a  # Will fail with cycle detection error
```

#### 3. Out of Disk Space

```bash
# Clean old builds if space is running low
$ chisel cleanup --aur --verbose

# Manually clean older builds (older than 14 days)
$ chisel cleanup --aur --max-age 14d
```

### Getting Help

For AUR-specific issues:

1. Check the build log: `/kod/build-logs/[package]-[version].log`
2. Visit the AUR page: `https://aur.archlinux.org/packages/[package]`
3. Check the PKGBUILD for known issues or comments
4. File an issue with the AUR maintainer if needed

## Advanced Features

### Custom Build Options

Current implementation uses standard makepkg options. Custom options can be passed via environment variables:

```bash
# Build with different compiler flags
CFLAGS="-O2 -march=native" chisel install some-aur-package
```

### Build Cache Management

Build cache is automatically managed:

```bash
# View build cache size
du -sh /kod/build-cache/

# Manual cleanup (remove builds older than X days)
find /kod/build-cache -maxdepth 1 -type d -mtime +7 -exec rm -rf {} \;
```

## Performance Notes

### RPC Caching

AUR RPC requests are cached locally with a 24-hour TTL:

```bash
# Force refresh of AUR cache
rm /path/to/chisel/aur.cache.json

# Then run any command that searches AUR
chisel search package-name
```

### Build Time

First-time builds take longer due to:
- PKGBUILD download and parsing
- Dependency installation
- Actual compilation

Subsequent installs of the same package are faster due to caching.

## Future Enhancements

Planned improvements for future releases:

- Binary caching for common AUR packages
- Parallel build support for multiple packages
- Custom build flags per package
- Build verification and validation
- Automated security scanning of PKGBUILDs

## References

- [AUR Official Documentation](https://wiki.archlinux.org/title/Arch_User_Repository)
- [PKGBUILD Guidelines](https://wiki.archlinux.org/title/PKGBUILD)
- [Making Packages](https://wiki.archlinux.org/title/Makepkg)
- [AUR RPC Interface](https://aur.archlinux.org/rpc/)

## Support

For issues or questions about Chisel's AUR integration:

1. Check this documentation
2. Review the build logs in `/kod/build-logs/`
3. File an issue on GitHub with:
   - Package name and version
   - Build log output
   - System information (uname -a)
   - Steps to reproduce
