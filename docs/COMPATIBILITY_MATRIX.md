# Compatibility Matrix

## Target Platforms

| Area | Target Minimum |
|---|---|
| Kernel | 5.x (best effort) |
| Distro | Ubuntu/Debian, Fedora/RHEL-family, Arch, openSUSE |
| Init | systemd (primary), non-systemd best-effort |
| Desktop | GNOME/KDE/Wayland/X11 (mostly independent) |
| Arch | amd64, arm64 |

## Command Compatibility Notes

### Clean
- User-level cleaning works on all supported distros.
- System-level cleaning requires root and may vary by package manager.

### Analyze
- Works on all distros; excludes `/proc`, `/sys`, `/dev`, `/run` by default.

### Purge
- Works on all distros; depends on filesystem access only.

### Status
- Uses gopsutil + `/proc` fallback; no root required.

### Optimize
- Requires `systemd` for journal vacuuming and resolver cache.
- Actions degrade gracefully when tools are missing.

### Uninstall
- Package manager integration depends on detected backend.
- Snap/Flatpak support depends on package presence.
