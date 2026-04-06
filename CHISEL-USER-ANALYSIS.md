# Chisel-User Analysis Summary

## (a) What is Chisel-User?

**Chisel-User** is a wrapper script and setup system that enables **user-level package management** without requiring root/sudo access. It is a transparent abstraction layer over the core chisel binary.

### Key Components:

1. **chisel-user** (bash wrapper script)
   - Simple wrapper that automatically detects and uses user-level configuration
   - Located at: `/home/abuss/Work/devel/chisel2/chisel-user`
   - ~126 lines of bash code
   - Fallback mechanism: user config → system config → defaults

2. **chisel-user-init.sh** (setup script)
   - One-time initialization script that sets up the user environment
   - Creates XDG-compliant directories:
     - `~/.local/share/chisel/` (data/store)
     - `~/.config/chisel/` (configuration)
     - `~/.local/bin/` (symlinks/executables)
   - Updates shell configuration files (~/.bashrc, ~/.zshrc, etc.)
   - Sets environment variables: CHISEL_USER_BASE_DIR, PATH, LD_LIBRARY_PATH

### How It Works:

```
User runs: chisel-user install vim
    ↓
chisel-user wrapper detects user config at ~/.config/chisel/config.json
    ↓
Calls: /path/to/chisel --base-dir ~/.local/share/chisel --config ~/.config/chisel/config.json install vim
    ↓
Same core chisel binary processes the command
    ↓
Packages installed to ~/.local/share/chisel/store/
    ↓
Symlinks created in ~/.local/bin/
```

### Configuration Priority:
1. Command-line flags (`--base-dir`, `--config`)
2. Environment variables (`CHISEL_USER_BASE_DIR`, `CHISEL_CONFIG`)
3. User config file (`~/.config/chisel/config.json`)
4. System config file (`/etc/chisel/config.json`)
5. Built-in defaults

---

## (b) AUR Support in Chisel-User

**YES - Chisel-User supports AUR packages.**

### How AUR is Integrated:

Chisel-User inherits **full AUR support** from the core chisel binary because it uses the same codebase. The chisel-user wrapper simply passes all commands through to chisel with user-level configuration.

### Supported AUR Commands (via chisel-user):

1. **chisel-user install <pkg>** - Installs packages, with automatic AUR fallback
   - Checks official repos first
   - Falls back to AUR if not found in official repositories
   - Automatically resolves mixed dependencies (official + AUR)
   - Builds AUR packages using makepkg

2. **chisel-user search <pattern>** - Searches both official repos and AUR
   - Official repos are searched first
   - AUR is searched as fallback

3. **chisel-user info <package>** - Shows package information from either source
   - Works for both official and AUR packages

4. **chisel-user upgrade [pkg]** - Upgrades packages including AUR ones
   - Detects AUR packages via Source field in registry
   - Checks for updates via AUR RPC API

5. **chisel-user cleanup [--aur]** - Cleanup with optional AUR build cache
   - **New feature**: `--aur` flag cleans AUR build cache and logs
   - Tracks build artifacts at `/kod/build-cache/`
   - Removes old build logs at `/kod/build-logs/`
   - Default retention: 7 days

### AUR Architecture Components (Inherited by chisel-user):

**Location**: `/home/abuss/Work/devel/chisel2/pkg/aur/`

Files:
- `rpc.go` - AUR RPC client for querying packages from https://aur.archlinux.org/rpc/
- `git.go` - Git handler for cloning PKGBUILD repositories
- `pkgbuild.go` - PKGBUILD parser for extracting metadata
- `types.go` - Data structures (AURPackage, PKGBUILDInfo)

Features:
- RPC caching with 24-hour TTL
- Rate limiting (4000 requests/day)
- Network error handling and timeouts
- Bash array parsing in PKGBUILDs
- Dependency extraction from build files

**Build System** (`/home/abuss/Work/devel/chisel2/pkg/build/`):
- makepkg integration for building AUR packages
- Persistent build cache at `/kod/build-cache/`
- Build logs saved to `/kod/build-logs/`
- Automatic cleanup based on age (configurable)

**Registry Enhancement**:
- Tracks package source: "official" vs "aur"
- Records repository information
- Maintains update dates
- Version history tracking

---

## (c) Relevant Code and Configuration

### chisel-user Script (lines 1-126):

**Global Options**:
```bash
--setup              Initialize user-level chisel environment
--base-dir <path>    Override user base directory
--config <path>      Override user config file
--help               Show help message
```

**Passes Through All Commands**:
All chisel commands work identically through chisel-user:
- sync, search, info, install, list, remove, upgrade, cleanup, cache

**Key Logic** (lines 108-126):
```bash
# Add user base directory if available
if [[ -n "$USER_BASE_DIR" ]]; then
    CHISEL_ARGS+=(--base-dir "$USER_BASE_DIR")
elif has_user_config; then
    CHISEL_ARGS+=(--base-dir "$CHISEL_USER_BASE_DIR")
fi

# Add user config if available
if [[ -n "$USER_CONFIG" ]]; then
    CHISEL_ARGS+=(--config "$USER_CONFIG")
elif has_user_config; then
    CHISEL_ARGS+=(--config "$CHISEL_USER_CONFIG")
fi

# Pass remaining arguments to chisel
exec "$CHISEL_BIN" "${CHISEL_ARGS[@]}" "$@"
```

### chisel-user-init.sh Configuration (lines 150-179):

Default user configuration created:
```json
{
  "base_dir": "~/.local/share/chisel",
  "symlink_root": "~/.local/bin",
  "mirror_url": "https://mirror.rackspace.com/archlinux",
  "architecture": "x86_64",
  "repositories": ["core", "extra", "community"],
  "verify_signatures": false,
  "max_concurrent_downloads": 5,
  "download_timeout": 300,
  "keep_versions": 3
}
```

### Directory Structure Created:

```
~/.local/share/chisel/
├── store/           # Extracted packages
├── db/             # Synced package databases
├── wrappers/       # Package wrapper scripts
├── cache/          # Downloaded packages
└── registry.json   # Installed packages registry

~/.config/chisel/
└── config.json     # User configuration

~/.local/bin/       # User symlinks (in PATH)
```

### Cleanup with AUR Support:

From `internal/cli/cleanup.go`:
```bash
# Standard cleanup (official packages only)
chisel-user cleanup

# Cleanup with AUR build cache
chisel-user cleanup --aur

# Verbose cleanup showing details
chisel-user cleanup --aur --verbose

# Preview without making changes
chisel-user cleanup --aur --dry-run

# Force cleanup without confirmation
chisel-user cleanup --aur --force
```

---

## Commands Supported by chisel-user

All chisel commands work through chisel-user wrapper:

### Phase 1 (Database & Search):
- `sync` - Sync package databases
- `search <pattern>` - Search packages (official + AUR fallback)
- `info <package>` - Show package information

### Phase 2 (Download & Extract):
- `download <pkg>` - Download packages
- `extract <file>` - Extract packages

### Phase 3 (Installation):
- `install <pkg>` - Install with dependencies (official or AUR)
- `remove <pkg>` - Remove packages
- `list` - List installed packages
- `list --verbose` - Detailed package information

### Phase 4 (Upgrades):
- `upgrade` - Upgrade all packages
- `upgrade <pkg>` - Upgrade specific packages
- `upgrade --dry-run` - Preview upgrades

### Phase 5 (Cleanup):
- `cleanup` - Remove old package versions
- `cleanup --aur` - Also clean AUR build cache (NEW)
- `cleanup --dry-run` - Preview cleanup
- `cleanup --force` - Force without confirmation
- `cleanup --verbose` - Show details
- `cache --list` - Show cache contents
- `cache --dry-run` - Preview cache clean
- `cache --force` - Clean all cached packages

---

## Key Differences: chisel vs chisel-user

| Feature | chisel | chisel-user |
|---------|--------|-----------|
| **Use Case** | System-wide (sudo required) | User-level (no sudo needed) |
| **Base Directory** | `/kod/` | `~/.local/share/chisel/` |
| **Symlink Location** | `/usr/bin/` | `~/.local/bin/` |
| **Config Location** | `/etc/chisel/config.json` | `~/.config/chisel/config.json` |
| **Permissions** | Root | Current user |
| **Isolation** | System-wide | Per-user |
| **AUR Support** | Yes (both support AUR equally) | Yes (same core) |
| **Setup** | Direct use | `chisel-user-init.sh` once |

---

## Installation & Limitations

### Limitations:

None specific to AUR. chisel-user inherits both capabilities and limitations from core chisel:

1. **makepkg Requirements**: Building AUR packages requires build tools (gcc, make, etc.)
2. **Build Cache**: AUR builds stored at `/kod/build-cache/` (system-level)
3. **Git Required**: For cloning PKGBUILD repositories
4. **Network Required**: RPC calls to aur.archlinux.org

### XDG Compliance:

chisel-user follows XDG Base Directory specification:
- Config: `$XDG_CONFIG_HOME/chisel/` (default: `~/.config/chisel/`)
- Data: `$XDG_DATA_HOME/chisel/` (default: `~/.local/share/chisel/`)
- Executables: `~/.local/bin/`

---

## Relationship to Core Chisel

**chisel-user is NOT a separate implementation** - it is a **configuration wrapper + setup utility** for the core chisel binary.

- **Single Codebase**: Uses the same Go binary (`/home/abuss/Work/devel/chisel2/cmd/chisel/main.go`)
- **No Duplication**: chisel-user is just bash wrapper (~126 lines)
- **Full Feature Parity**: Every chisel feature works through chisel-user
- **Transparent**: Users run `chisel-user` but core logic is identical

The architecture is:
```
User Interface Layer:
- chisel-user (bash wrapper) → detects user config → calls chisel
- chisel (bash wrapper) → uses default config → calls chisel

Core Implementation:
- Single Go binary (cmd/chisel/main.go)
- All commands: sync, install, upgrade, cleanup, cache, etc.
- AUR support: pkg/aur/, pkg/build/
- Registry: pkg/registry/ with AUR fields
```

---

## Example: Installing an AUR Package

```bash
# One-time setup
./chisel-user-init.sh

# Reload shell
source ~/.bashrc

# Install AUR package (e.g., yay)
chisel-user install yay

# What happens internally:
# 1. chisel-user wrapper calls:
#    chisel --base-dir ~/.local/share/chisel \
#           --config ~/.config/chisel/config.json \
#           install yay

# 2. chisel checks official repos first (not found)
# 3. chisel queries AUR RPC for "yay" (found)
# 4. chisel clones PKGBUILD from AUR
# 5. chisel builds with makepkg
# 6. chisel extracts to ~/.local/share/chisel/store/yay/
# 7. chisel creates symlink ~/.local/bin/yay
# 8. chisel records in ~/.local/share/chisel/registry.json with source="aur"

# Clean up including AUR build cache
chisel-user cleanup --aur --force
```

