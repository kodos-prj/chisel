# Configuration Guide

Packmgr supports multiple ways to specify configuration, giving you flexibility for different use cases.

## Configuration Priority

When packmgr loads configuration, it uses the following priority order (highest to lowest):

1. **Command-line flags** - Highest priority, overrides everything
2. **Environment variables** - Second priority  
3. **Configuration file** - Third priority
4. **Built-in defaults** - Lowest priority, used if nothing else is specified

## Configuration Methods

### 1. Command-Line Flags

Override specific configuration values on the command line:

```bash
# Use custom config file
packmgr -c /path/to/config.json sync
packmgr --config /path/to/config.json sync

# Override base directory
packmgr --base-dir /tmp/packmgr-test sync

# Override mirror URL
packmgr --mirror https://mirrors.kernel.org/archlinux sync

# Combine multiple options
packmgr -c myconfig.json --base-dir /tmp/test --mirror https://example.com/archlinux search vim
```

**Available global flags:**
- `-c, --config <path>` - Path to configuration file
- `--base-dir <path>` - Base directory for all packmgr data (overrides config)
- `--mirror <url>` - Arch mirror URL (overrides config)

### 2. Environment Variables

Set configuration via environment variables:

```bash
# Set custom config file path
export PACKMGR_CONFIG=/home/user/.packmgr.json
packmgr sync

# Set custom base directory
export PACKMGR_BASE_DIR=/opt/packmgr
packmgr sync

# Use for single command
PACKMGR_BASE_DIR=/tmp/test packmgr sync --status
```

**Available environment variables:**
- `PACKMGR_CONFIG` - Path to configuration file
- `PACKMGR_BASE_DIR` - Base directory for packmgr data

### 3. Configuration File

Create a JSON configuration file with your settings.

**Default location:** `/etc/packmgr/config.json`

**Example configuration:**

```json
{
  "base_dir": "/kod",
  "symlink_root": "/",
  "mirror_url": "https://mirror.rackspace.com/archlinux",
  "architecture": "x86_64",
  "repositories": ["core", "extra"],
  "verify_signatures": false,
  "max_concurrent_downloads": 5,
  "download_timeout": 300,
  "keep_versions": 3
}
```

## Configuration Options

### Directory Paths

#### `base_dir` (string)
- **Default:** `/kod`
- **Description:** Base directory for all packmgr data
- **Example:** `/kod`, `/opt/packmgr`, `/home/user/.local/packmgr`

When you set `base_dir`, all the following paths are automatically derived:
- Store: `{base_dir}/store`
- Registry: `{base_dir}/registry.json`
- Databases: `{base_dir}/db`
- Wrappers: `{base_dir}/wrappers`
- Cache: `{base_dir}/cache`

#### `symlink_root` (string)
- **Default:** `/`
- **Description:** Root directory where symlinks to binaries are created
- **Example:** `/`, `/usr/local`

#### `store_root` (string)
- **Default:** `{base_dir}/store`
- **Description:** Directory where extracted packages are stored
- **Note:** Usually auto-derived from `base_dir`

#### `wrapper_dir` (string)
- **Default:** `{base_dir}/wrappers`
- **Description:** Directory for wrapper scripts
- **Note:** Usually auto-derived from `base_dir`

#### `db_path` (string)
- **Default:** `{base_dir}/db`
- **Description:** Directory for synced Arch package databases
- **Note:** Usually auto-derived from `base_dir`

#### `cache_path` (string)
- **Default:** `{base_dir}/cache`
- **Description:** Directory for downloaded package files (.pkg.tar.zst)
- **Note:** Usually auto-derived from `base_dir`

#### `registry_path` (string)
- **Default:** `{base_dir}/registry.json`
- **Description:** Path to the package registry file
- **Note:** Usually auto-derived from `base_dir`

### Mirror Configuration

#### `mirror_url` (string)
- **Default:** `https://mirror.rackspace.com/archlinux`
- **Description:** Base URL for Arch Linux mirror
- **Examples:**
  - `https://mirror.rackspace.com/archlinux`
  - `https://mirrors.kernel.org/archlinux`
  - `https://mirror.f4st.host/archlinux`
  - `https://geo.mirror.pkgbuild.com` (geo-distributed CDN)

**Popular Arch mirrors:**
- US: `https://mirror.rackspace.com/archlinux`
- US: `https://mirrors.kernel.org/archlinux`
- Global CDN: `https://geo.mirror.pkgbuild.com`
- See full list: https://archlinux.org/mirrors/status/

#### `architecture` (string)
- **Default:** `x86_64`
- **Description:** Target CPU architecture
- **Options:** `x86_64`, `aarch64` (ARM 64-bit)

#### `repositories` (array of strings)
- **Default:** `["core", "extra"]`
- **Description:** List of Arch repositories to sync
- **Available repositories:**
  - `core` - Essential packages
  - `extra` - Additional packages  
  - `multilib` - 32-bit libraries on 64-bit systems
  - `community` - Community-maintained packages (deprecated in Arch, merged into extra)

**Example:**
```json
{
  "repositories": ["core", "extra", "multilib"]
}
```

### Download Settings

#### `download_timeout` (integer)
- **Default:** `300`
- **Description:** Download timeout in seconds
- **Example:** `300` (5 minutes), `600` (10 minutes)

#### `max_concurrent_downloads` (integer)
- **Default:** `5`
- **Description:** Maximum number of concurrent package downloads
- **Range:** 1-20 recommended
- **Example:** `5`, `10`

### Security Settings

#### `verify_signatures` (boolean)
- **Default:** `false`
- **Description:** Whether to verify package GPG signatures
- **Note:** Currently optional for simplicity. Will be enabled by default in v1.0

### Maintenance Settings

#### `keep_versions` (integer)
- **Default:** `3`
- **Description:** Number of old package versions to keep during cleanup
- **Example:** `3` (keep last 3 versions), `1` (keep only current)

## Example Configurations

### Development/Testing Setup

For local development or testing without requiring root access:

```json
{
  "base_dir": "/home/user/.local/packmgr",
  "symlink_root": "/home/user/.local",
  "mirror_url": "https://mirrors.kernel.org/archlinux",
  "repositories": ["core", "extra"],
  "download_timeout": 600
}
```

**Usage:**
```bash
packmgr -c ~/.packmgr-dev.json sync
packmgr -c ~/.packmgr-dev.json install vim
```

### Production Setup

System-wide installation with default paths:

```json
{
  "base_dir": "/kod",
  "symlink_root": "/",
  "mirror_url": "https://geo.mirror.pkgbuild.com",
  "architecture": "x86_64",
  "repositories": ["core", "extra", "multilib"],
  "verify_signatures": true,
  "max_concurrent_downloads": 10,
  "download_timeout": 300,
  "keep_versions": 2
}
```

**Usage:**
```bash
sudo packmgr sync
sudo packmgr install vim
```

### Temporary Testing

Quick testing without creating a config file:

```bash
# Test in temporary directory
packmgr --base-dir /tmp/packmgr-test sync
packmgr --base-dir /tmp/packmgr-test search vim

# Test with different mirror
packmgr --base-dir /tmp/test --mirror https://mirror.f4st.host/archlinux sync
```

### Environment-Based Configuration

Use environment variables for containerized environments:

```bash
# In Docker/container
export PACKMGR_BASE_DIR=/opt/packmgr
export PACKMGR_CONFIG=/etc/packmgr/prod.json

packmgr sync
packmgr install package-name
```

## Configuration File Locations

Packmgr looks for configuration in the following locations (in order):

1. Path specified with `-c` or `--config` flag
2. Path in `PACKMGR_CONFIG` environment variable
3. `/etc/packmgr/config.json` (default)
4. Built-in defaults (if no config file exists)

## Creating a Configuration File

### Method 1: Manual Creation

Create `/etc/packmgr/config.json`:

```bash
sudo mkdir -p /etc/packmgr
sudo nano /etc/packmgr/config.json
```

Paste your JSON configuration and save.

### Method 2: Generate from Defaults

Generate a config file with default values:

```go
// Future feature - config generation command
packmgr config init
packmgr config init --output /home/user/.packmgr.json
```

(Note: This feature is planned for a future release)

### Method 3: Programmatic Generation

Use Go code to generate a config:

```go
package main

import (
	"github.com/yourusername/packmgr-go/pkg/config"
)

func main() {
	cfg := config.DefaultConfig()
	cfg.MirrorURL = "https://mirrors.kernel.org/archlinux"
	cfg.Repositories = []string{"core", "extra", "multilib"}
	
	if err := cfg.Save("/etc/packmgr/config.json"); err != nil {
		panic(err)
	}
}
```

## Validation

Packmgr automatically validates configuration on load:

- Sets defaults for missing fields
- Ensures all paths are properly derived from `base_dir`
- Validates repository names
- Checks numeric ranges

If configuration is invalid, packmgr will:
1. Print an error message
2. Fall back to built-in defaults
3. Continue execution (graceful degradation)

## Tips

1. **Start with defaults:** Use built-in defaults for initial testing
2. **Override sparingly:** Only override what you need to change
3. **Test configurations:** Use `--base-dir /tmp/test` to test configs safely
4. **Use environment variables in CI/CD:** Easy to configure without files
5. **Keep configs in version control:** Track changes to configuration
6. **Use different configs per environment:** dev.json, staging.json, prod.json

## Troubleshooting

### Configuration not loading

Check if config file exists and is valid JSON:
```bash
cat /etc/packmgr/config.json
python -m json.tool /etc/packmgr/config.json
```

### Override not working

Remember priority order:
1. CLI flags override everything
2. Environment variables override config file
3. Config file overrides defaults

### Permission denied

If using `/kod` or `/etc/packmgr`, you need root access:
```bash
sudo packmgr sync
```

Or use a user-writable location:
```bash
packmgr --base-dir ~/.local/packmgr sync
```

## See Also

- [Main Documentation](../README.md)
- [Implementation Plan](../docs/03-IMPLEMENTATION-PLAN.md)
- [Architecture Specification](../docs/00-SPECIFICATION.md)
