# Chisel User-Level Package Manager Guide

This guide explains how to use chisel as a regular user to install and manage packages at the user level without requiring root/sudo access.

## Overview

By default, chisel stores packages in `/kod/` which requires root access. This guide shows how to configure chisel to use user-level directories following the XDG Base Directory specification, allowing any user to independently manage packages.

**Key Benefits:**
- ✅ No root/sudo required
- ✅ Packages isolated per user
- ✅ Follows XDG standards (~/.local, ~/.config)
- ✅ Easy to set up and tear down
- ✅ Compatible with all chisel features

## Quick Start

### 1. Initialize User Environment

Run the setup script to automatically configure chisel for user-level use:

```bash
./chisel-user-init.sh
```

This script:
- Creates user directories (~/.local/share/chisel, ~/.config/chisel, etc.)
- Creates a user configuration file
- Updates your shell configuration (PATH, LD_LIBRARY_PATH, environment variables)

**Output:**
```
✓ Chisel binary found
✓ Creating user-level directories...
✓ Creating user-level configuration...
✓ Updating shell configuration...
```

### 2. Reload Your Shell

```bash
# For bash
source ~/.bashrc

# For zsh
source ~/.zshrc

# For fish
source ~/.config/fish/config.fish
```

### 3. Start Using Chisel

```bash
# Sync package databases
chisel-user sync

# Search for a package
chisel-user search vim

# Install a package
chisel-user install nano

# List installed packages
chisel-user list

# Remove a package
chisel-user remove nano
```

## User-Level Directory Structure

When you run the setup script, chisel creates this directory structure:

```
~/.local/share/chisel/              # Main data directory (CHISEL_USER_BASE_DIR)
├── store/                          # Extracted packages
├── db/                             # Synced package databases
├── wrappers/                       # Package wrapper scripts
├── cache/                          # Downloaded packages
└── registry.json                   # Installed packages registry

~/.config/chisel/                   # Configuration directory
└── config.json                     # User configuration

~/.local/bin/                       # User symlinks (in PATH)
```

## How It Works

### Configuration Priority

Chisel uses this priority order for configuration:

1. **Command-line flags** (`--base-dir`, `--config`)
2. **Environment variables** (`CHISEL_USER_BASE_DIR`, `CHISEL_CONFIG`, `CHISEL_BASE_DIR`)
3. **User config file** (`~/.config/chisel/config.json`)
4. **System config file** (`/etc/chisel/config.json`)
5. **Built-in defaults** (`/kod`, `/etc/chisel/config.json`)

### Using chisel-user vs chisel

**chisel-user** (recommended for users):
- Automatically uses user-level configuration
- Seamless user experience
- Falls back to system config if needed

```bash
chisel-user install nano
```

**chisel** (with environment variables):
- Full control
- Flexible for different use cases

```bash
export CHISEL_USER_BASE_DIR=~/.local/share/chisel
export CHISEL_CONFIG=~/.config/chisel/config.json
chisel install nano
```

**Direct flag usage:**
```bash
chisel \
  --base-dir ~/.local/share/chisel \
  --config ~/.config/chisel/config.json \
  install nano
```

## Environment Variables

After setup, your shell will have:

```bash
# User-level base directory
export CHISEL_USER_BASE_DIR="$HOME/.local/share/chisel"

# Add user bin to PATH (for package executables)
export PATH="$PATH:$HOME/.local/bin"

# Add user lib to LD_LIBRARY_PATH (for package libraries)
export LD_LIBRARY_PATH="$HOME/.local/lib:$LD_LIBRARY_PATH"
```

You can override these:

```bash
# Use a different base directory
export CHISEL_USER_BASE_DIR=/path/to/packages

# Use a different symlink location
export CHISEL_SYMLINK_DIR=$HOME/custom/bin
```

## Common Operations

### Search and Install

```bash
# Search for packages
chisel-user search vim

# Get detailed package information
chisel-user info vim

# Install with all dependencies automatically resolved
chisel-user install vim

# Install without extracting (if already in store)
chisel-user install vim --no-extract
```

### Manage Installed Packages

```bash
# List all packages
chisel-user list

# List with detailed information
chisel-user list --verbose

# Remove a package
chisel-user remove vim

# Remove without confirmation
chisel-user remove vim --force
```

### Updates and Cleanup

```bash
# Check for updates
chisel-user upgrade --dry-run

# Upgrade all packages
chisel-user upgrade

# Upgrade specific packages
chisel-user upgrade bash curl

# Remove old versions (keeps 3 most recent)
chisel-user cleanup --dry-run

# Remove old versions without confirmation
chisel-user cleanup --force

# Preview cleanup in verbose mode
chisel-user cleanup --verbose --dry-run
```

### Cache Management

```bash
# Show cache contents
chisel-user cache --list

# Preview cache clean
chisel-user cache --dry-run

# Clean all cached packages
chisel-user cache --force
```

## Troubleshooting

### "Command not found: chisel-user"

Make sure you:
1. Ran `./chisel-user-init.sh`
2. Reloaded your shell (`source ~/.bashrc`)
3. Have ~/.local/bin in your PATH

Check:
```bash
echo $PATH
ls -la ~/.local/bin/chisel-user
```

### Setup script not found

The setup script must be in the same directory as the chisel binary:

```bash
ls -la chisel*
# Should show: chisel, chisel-user, chisel-user-init.sh
```

### Packages not in PATH

After installation, packages should be available in ~/.local/bin:

```bash
# Check if symlinks were created
ls ~/.local/bin/

# Verify PATH includes ~/.local/bin
echo $PATH
```

If packages still aren't available:
1. Reload your shell: `source ~/.bashrc`
2. Check LD_LIBRARY_PATH: `echo $LD_LIBRARY_PATH`
3. Verify package was installed: `chisel-user list`

### Database sync fails

Make sure you have internet connectivity:

```bash
# Try syncing with verbose output
chisel-user sync --verbose

# Check if you can reach the mirror
curl -I https://mirror.rackspace.com/archlinux/core/os/x86_64/core.db
```

### Installation fails with permission errors

User-level chisel should not require sudo. If you get permission errors:

1. Check directory permissions:
```bash
ls -la ~/.local/share/chisel/
```

2. Ensure user owns the directories:
```bash
chown -R $USER ~/.local/share/chisel
chown -R $USER ~/.config/chisel
```

3. Check disk space:
```bash
df -h ~/.local/share/
```

### Configuration conflicts

If you have both user and system configs:

1. User config takes priority (good for customization)
2. To force system config: `chisel --base-dir /kod install vim`
3. To use specific config: `chisel --config /path/to/config.json install vim`

## Advanced Usage

### Multiple Users

Each user can run the setup independently:

```bash
# User 1
user1@host:~$ ./chisel-user-init.sh

# User 2 (different user)
user2@host:~$ ./chisel-user-init.sh

# Each has isolated packages
user1@host:~$ chisel-user list
user2@host:~$ chisel-user list    # Different packages
```

### Custom Base Directories

Override the user base directory:

```bash
# Use a different location
export CHISEL_USER_BASE_DIR=/tmp/my-packages
chisel-user sync

# Or use the wrapper with explicit path
chisel-user --base-dir /tmp/my-packages install vim
```

### Temporary Package Testing

Test packages without affecting your main installation:

```bash
# Create temporary directory
mkdir /tmp/chisel-test
export CHISEL_USER_BASE_DIR=/tmp/chisel-test

# Install and test
chisel-user sync
chisel-user install test-package

# Clean up
rm -rf /tmp/chisel-test
```

### Integration with Other Tools

Since packages are installed to ~/.local/bin, you can:

```bash
# Add to PATH temporarily
export PATH=$PATH:$HOME/.local/bin

# Use in scripts
#!/bin/bash
export PATH=$PATH:$HOME/.local/bin
my-package-command

# Add to systemd user services
# ~/.config/systemd/user/my-service.service
[Service]
Environment="PATH=%h/.local/bin:%h/.local/lib"
ExecStart=%h/.local/bin/my-package
```

## Migration from System-Level

If you have packages installed system-wide and want to migrate to user-level:

```bash
# Initialize user environment
./chisel-user-init.sh

# Reinstall packages in user location
chisel-user sync
chisel-user install package1 package2 package3

# Verify new installation
chisel-user list --verbose

# Remove system-level packages (requires sudo)
sudo chisel remove package1 package2 package3
```

## Cleanup and Uninstall

### Remove Individual Packages

```bash
chisel-user remove package-name
```

### Remove All User Data

```bash
# Remove all chisel user data
rm -rf ~/.local/share/chisel
rm -rf ~/.config/chisel

# Remove symlinks
rm -rf ~/.local/bin/*

# Remove from shell config (manually edit ~/.bashrc, ~/.zshrc, etc.)
# Remove lines added by the setup script
```

## Best Practices

### 1. Regular Cleanup

```bash
# Periodically remove old versions
chisel-user cleanup --dry-run    # Preview first
chisel-user cleanup --force      # Then execute
```

### 2. Cache Management

```bash
# Clean cache after upgrades
chisel-user cache --list         # See what's there
chisel-user cache --force        # Remove all
```

### 3. Database Updates

```bash
# Keep databases fresh
chisel-user sync                # Sync databases regularly
```

### 4. Monitoring Usage

```bash
# Check disk usage
du -sh ~/.local/share/chisel/

# List packages and sizes
chisel-user list --verbose
```

## FAQ

**Q: Can multiple users share packages?**
A: By default, each user has isolated packages. To share, set the same CHISEL_USER_BASE_DIR and ensure read permissions.

**Q: Do I need sudo for anything?**
A: No. User-level chisel is completely sudo-free. All packages go to user-owned directories.

**Q: Can I use the system chisel and chisel-user together?**
A: Yes. System chisel uses /kod, user-level uses ~/.local/share/chisel. They don't conflict.

**Q: What if ~/.local/bin is not in my PATH?**
A: The setup script adds it. If not, manually add to your ~/.bashrc:
```bash
export PATH="$PATH:$HOME/.local/bin"
```

**Q: Can I move packages after installation?**
A: You can move the entire ~/.local/share/chisel directory. Symlinks will need to be recreated.

**Q: What about LD_LIBRARY_PATH?**
A: Some packages need libraries from their own store. Set LD_LIBRARY_PATH to enable this:
```bash
export LD_LIBRARY_PATH="$HOME/.local/lib:$LD_LIBRARY_PATH"
```

## See Also

- [Chisel Main Documentation](./README.md)
- [Test Workflow Guide](./TEST-WORKFLOW.md)
- [Quick Test Guide](./QUICK-TEST.md)
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)

## Getting Help

```bash
# Show help for chisel-user
chisel-user --help

# Show help for setup
./chisel-user-init.sh --help

# Show chisel help
chisel help
```
