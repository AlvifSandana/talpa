# PRD — Talpa (Linux Universal)
**Dokumen:** Product Requirements Document (PRD)  
**Produk:** Talpa — *Deep clean & optimize for Linux*  
**Inspirasi:** [`tw93/Mole`](https://github.com/tw93/Mole) (CLI/TUI all‑in‑one: clean, uninstall, optimize, analyze, status, purge, installer)  
**Owner:** KucingSakti  
**Status:** Draft v0.1  
**Tanggal:** 2026-02-16 (Asia/Jakarta)

---

## 1) Ringkasan
Talpa adalah **CLI/TUI single-binary** untuk Linux (universal lintas distro) yang membantu pengguna:
- **Membersihkan** cache/log/temp & residu browser/dev-tools
- **Menghapus aplikasi + leftover** (config/data/desktop entry) secara aman
- **Optimasi** sistem (aksi aman & reversible jika memungkinkan)
- **Menganalisis disk** secara interaktif (tree explorer)
- **Monitoring** kesehatan sistem real-time
- **Purge** artefak build project (node_modules, target, venv, dll)
- **Bersih-bersih installer** (.deb/.rpm/.AppImage/.tar.*)

Talpa mengadopsi prinsip Mole: **safety-first**, ada **dry-run**, **whitelist**, serta **operation log** untuk audit/troubleshooting.

---

## 2) Latar Belakang & Problem
Di Linux (terutama workstation dev), konsumsi disk membengkak dari:
- Cache browser & aplikasi (Chromium/Firefox, Electron apps, dsb.)
- Cache dev tools (npm/pnpm/yarn, pip, cargo, gradle, maven, go build cache, docker)
- Log/system journal membengkak, file temp, thumbnail cache, trash
- Aplikasi dihapus tapi meninggalkan **sisa config/data** di XDG paths
- Banyak proyek lama menyimpan **artefak build** yang besar

Tool yang ada sering **tersebar**: `ncdu`, `bleachbit`, `stacer`, `journalctl --vacuum-*`, `apt clean`, dsb. Talpa menyatukan workflow itu jadi **satu tool** yang konsisten, aman, dan enak dipakai via TUI.

---

## 3) Tujuan (Goals)
1. **Reclaim disk space** secara signifikan dengan langkah aman & terukur.
2. **Satu workflow** untuk cleaning/uninstall/analyze/status/purge/installer.
3. **Universal Linux**: berjalan di distro mayor tanpa konfigurasi rumit.
4. **Safety & security by design**: validasi path, anti-symlink-traversal, proteksi direktori kritis, konfirmasi berlapis untuk aksi destruktif.
5. **Fast**: scanning efisien (concurrency, cache, limit depth), dan robust (timeout pada mount/network FS).

### Non-Goals
- Bukan pengganti backup/restore (Timeshift, btrfs snapshots).
- Bukan “tweak” agresif yang merusak sistem (mis. menghapus file sistem tanpa whitelist).
- Tidak mengubah konfigurasi kernel/bootloader secara otomatis.
- Tidak melakukan “auto-clean” tanpa konfirmasi (kecuali mode scripted yang eksplisit).

---

## 4) Target User & Persona
1. **Developer** (Node/Laravel/Python/Go/Rust) dengan banyak repo & artefak build.
2. **Power user Linux desktop** (GNOME/KDE) yang butuh disk analyzer & cleanup rutin.
3. **Ops/DevOps ringan** pada laptop/VM (banyak docker image/volume, log menumpuk).

---

## 5) Positioning & Value Proposition
> “Seperti Mole di macOS, tapi khusus Linux: satu binary, satu UI, aman, dan ngerti ekosistem cache/leftover Linux.”

---

## 6) Scope Rilis (Phasing)
### MVP (v0.x)
- `talpa clean` (user-level + opsi system-level terbatas)
- `talpa analyze` (disk explorer)
- `talpa purge` (project build artifacts)
- `talpa status` (basic metrics)
- `talpa completion`, `talpa update`, `talpa remove`
- Safety core: dry-run, whitelist, blocked paths, operations.log

### v1.0
- `talpa uninstall` (paket manager + leftover XDG)
- `talpa installer`
- `talpa optimize` (aksi aman lintas distro)
- Integrasi distro: apt/dnf/pacman/zypper + snap/flatpak

### v1.x+
- Plugin ruleset (community contributions)
- “Profiles” (Dev, Desktop, Minimal)
- Integrasi launcher (rofi/wofi/fzf) & desktop search (opsional)

---

## 7) UX & Command Surface
### 7.1 Mode Interaktif (default)
Menjalankan `talpa` tanpa argumen membuka menu TUI:
- Clean
- Uninstall
- Optimize
- Analyze Disk
- Status
- Purge Projects
- Installer Cleanup
- Settings (whitelist, paths)
- Help / About

**Navigasi:** Arrow + Vim keys (`h/j/k/l`), search filter, multi-select, confirm dialog.

### 7.2 CLI Commands (paritas dengan Mole)
```
talpa            # interactive menu
talpa clean      # deep cleanup
talpa uninstall  # remove apps + leftovers
talpa optimize   # refresh caches/services
talpa analyze    # visual disk explorer
talpa status     # live system dashboard
talpa purge      # purge build artifacts
talpa installer  # remove installer files
talpa completion # shell completion
talpa update     # self-update
talpa remove     # uninstall talpa
```

### 7.3 Global Flags
- `--dry-run` : preview tanpa delete/modify
- `--debug` : log detail + klasifikasi risiko item
- `--yes` : non-interactive confirm (hanya untuk mode scripted)
- `--json` : output mesin (untuk CI/ops)
- `--no-oplog` : matikan operation log

---

## 8) Functional Requirements (FR)

### FR-1: Deep Cleanup (`talpa clean`)
**Tujuan:** menghapus file yang aman untuk dibuang agar reclaim disk.

#### 8.1 Kategori Cleanup (User-level, no sudo)
- **XDG caches:** `~/.cache/*`
- **Browser caches:** Chromium/Chrome/Brave/Vivaldi/Firefox cache dirs
- **Electron app caches:** Slack/Discord/VSCode caches (via XDG paths)
- **Dev tool caches (aman & default):**
  - Node: npm/yarn/pnpm cache (bukan `node_modules`)
  - Python: pip cache, poetry cache, `__pycache__` (opsional)
  - Rust: cargo registry cache (bukan toolchain/bin)
  - Java: gradle caches (selektif), maven `.m2/repository` (opsional dengan threshold & whitelist)
  - Go: `GOCACHE`, `GOMODCACHE` (opsional)
- **Trash:** `~/.local/share/Trash/*`
- **Thumbnails:** `~/.cache/thumbnails` (GNOME/KDE)
- **Logs user:** `~/.local/state` & app logs di `~/.local/share`

#### 8.2 Kategori Cleanup (System-level, butuh root, opt-in)
- Paket manager cache:
  - APT: `/var/cache/apt/archives`
  - DNF: `/var/cache/dnf`
  - Pacman: `/var/cache/pacman/pkg`
  - Zypper: `/var/cache/zypp`
- `systemd-journald` vacuum (berdasarkan size/age)
- `/tmp`, `/var/tmp` (dengan rule aman)

#### 8.3 Perilaku & Safety
- Semua kandidat ditampilkan dengan: **path**, **size**, **kategori**, **risk level** (low/med/high), **last modified**.
- Default: hanya item **low/med** dipilih otomatis; **high** harus dipilih manual.
- `--dry-run` menampilkan rencana aksi + estimasi space freed.
- `--whitelist` untuk proteksi path (exact match + glob terbatas).
- Setiap delete melewati **path validator** + **symlink guard** + **blocked paths**.

**Acceptance Criteria**
- Talpa bisa memindai ≤ 60 detik pada home directory 200–500GB (tanpa network mounts).
- Tidak pernah menghapus path terlarang (lihat SR-1).

---

### FR-2: Smart Uninstaller (`talpa uninstall`)
**Tujuan:** hapus aplikasi + sisa file yang umum tertinggal.

#### 8.4 Target Instalasi
- System packages: apt/dnf/pacman/zypper
- Flatpak: `flatpak uninstall --delete-data`
- Snap: `snap remove --purge`
- AppImage: hapus file `.AppImage` + leftover (berdasarkan desktop entry & folder config)

#### 8.5 Identifikasi Aplikasi
- Membaca `.desktop` entries dari:
  - `/usr/share/applications`
  - `~/.local/share/applications`
- Korelasikan ke:
  - package name (jika tersedia)
  - exec binary path
  - app id/desktop file name

#### 8.6 Leftover Rules (XDG-centric)
Untuk app yang di-uninstall, Talpa mencari kandidat leftover di:
- `~/.config/<app>`
- `~/.local/share/<app>`
- `~/.cache/<app>`
- `~/.local/state/<app>`
Dengan strategi aman:
- Minimal panjang nama ≥ 3 karakter
- Pencocokan bertahap: exact → normalized (lowercase, dash/underscore) → curated aliases
- Orphan rule (default): data dianggap orphan jika app sudah tidak terpasang **dan** file tidak berubah ≥ N hari (mis. 60 hari). Untuk `talpa uninstall`, orphan rule bisa di-bypass karena user explicitly uninstall.

**Acceptance Criteria**
- Menampilkan daftar aplikasi dengan estimasi total size (app + leftover).
- Hapus via package manager jika applicable, lalu bersihkan leftover terpilih.

---

### FR-3: System Optimization (`talpa optimize`)
**Tujuan:** merapikan dan menyegarkan service/cache agar sistem terasa lebih “sehat”.

#### 8.7 Aksi Optimize (safe, cross-distro)
- `systemd-journal` vacuum (previewable)
- Rebuild font cache (fontconfig)
- Update desktop database (mime/desktop)
- Clear resolver cache (jika systemd-resolved)
- Restart non-critical user services (opsional per DE)
- Refresh shell caches (GNOME/KDE) dengan deteksi environment

Semua aksi:
- Mendukung `--dry-run` dan `--whitelist` berbasis “rule id”
- Memiliki timeout & fallback bila tool tidak ada

**Acceptance Criteria**
- Optimize tidak berjalan jika mendeteksi kondisi berisiko (mis. low battery laptop, filesystem read-only, atau sedang ada upgrade paket berjalan).
- Semua aksi tercatat di operations.log.

---

### FR-4: Disk Analyzer (`talpa analyze [path]`)
**Tujuan:** disk explorer interaktif seperti “ncdu namun terintegrasi”.

#### 8.8 Perilaku
- Default root: `~` (home)
- Exclude: `/proc`, `/sys`, `/dev`, `/run` dan mount network (NFS/SMB) kecuali user specify.
- Menampilkan tree: size, persentase, last modified bucket.
- Action:
  - Open (xdg-open)
  - Reveal (print path)
  - Delete to Trash (default)
  - Permanent delete (double confirm)

**Acceptance Criteria**
- Delete via Trash untuk mengurangi risiko; permanent delete butuh 2 langkah konfirmasi.

---

### FR-5: Live Status (`talpa status`)
**Tujuan:** dashboard real-time untuk diagnosis performa.

#### 8.9 Metrics Minimal
- CPU: usage, load average
- Memory: used/available, swap
- Disk: usage, read/write throughput, top mount points
- Network: up/down throughput, IP
- Top processes by CPU/mem

#### 8.10 Health Score (opsional v1.0)
Skor komposit dari CPU, memory pressure, disk fullness, temperature (jika tersedia), I/O.

**Acceptance Criteria**
- Refresh interval configurable (default 1s).
- Works tanpa root.

---

### FR-6: Project Artifact Purge (`talpa purge`)
**Tujuan:** hapus artefak build besar dari folder project.

#### 8.11 Deteksi Artefak (default)
- Node: `node_modules`, `.next`, `dist`, `build`
- Rust: `target`
- Java: `target`, `build`, `.gradle`
- Python: `.venv`, `venv`, `__pycache__`, `.pytest_cache`
- Go: `vendor` (opsional), `bin` build output (opsional)
- Mobile/others: `.dart_tool`, `Pods`, `DerivedData` (jika ada)

#### 8.12 Scan Paths
- Default: `~/Projects`, `~/Code`, `~/dev`, `~/src`, `~/GitHub` (auto-detect jika ada)
- Custom: `talpa purge --paths` (menulis `~/.config/talpa/purge_paths`)

#### 8.13 Perilaku Aman
- Project “recent” (mis. modified < 7 hari) ditandai dan tidak dipilih default.
- Preview size per item; multi-select; confirm.

**Acceptance Criteria**
- Scan depth configurable (default 4, max 6) untuk performa.
- Tahan symlink loop.

---

### FR-7: Installer Cleanup (`talpa installer`)
**Tujuan:** menemukan “installer files” besar yang tercecer.

#### 8.14 File Types
- `.deb`, `.rpm`, `.pkg.tar.*`, `.AppImage`, `.run`
- archive: `.zip`, `.tar.gz`, `.tar.xz` (hanya yang matching heuristik installer)
- lokasi: `~/Downloads`, `~/Desktop`, cache manager (opsional)

**Acceptance Criteria**
- Label “source” (Downloads/Desktop/Package Cache/Other) agar user paham asalnya.

---

### FR-8: Update & Self-Manage
- `talpa update`:
  - download release terbaru dari GitHub
  - verifikasi checksum (SHA256) & signature (jika ada)
  - dukung `--force` untuk reinstall
- `talpa remove`:
  - hapus binary + config opsional (`--purge`)

---

## 9) Safety & Security Requirements (SR)
### SR-1: Path Validation & Blocklist
- Reject:
  - empty path
  - path mengandung control chars / null byte
  - traversal `/../`
  - symlink yang mengarah keluar scope atau ke system-critical path
- Blocked paths (minimal):
  - `/`, `/boot`, `/bin`, `/sbin`, `/lib`, `/lib64`, `/usr`, `/etc`, `/var` *(kecuali whitelist rule khusus & eksplisit untuk cache)*
  - `/proc`, `/sys`, `/dev`, `/run`

### SR-2: Sudo Minimization
- Operasi root hanya untuk kategori tertentu (apt cache, journal vacuum, /tmp tertentu).
- Prefer “read-only scan” tanpa root.

### SR-3: Timeouts & Network FS
- Deteksi mount type; network mounts diberi timeout dan skip by default.
- Semua external commands memakai timeout wrapper.

### SR-4: No Sensitive Data
- Talpa tidak mengumpulkan telemetry default.
- Log hanya menyimpan operasi file (path, size, action) — tanpa isi file.

---

## 10) Config, State, Logging
### 10.1 Direktori Konfig
`~/.config/talpa/`:
- `whitelist` (paths terlindungi)
- `purge_paths`
- `settings.json` (prefs UI)
- `operations.log` (audit)

### 10.2 Operation Log
- Format: JSON lines (mudah diparse) + human readable summary
- Env var: `TALPA_NO_OPLOG=1` untuk disable
- Setiap aksi delete/modify harus ada log entry.

---

## 11) Packaging & Distribution
### 11.1 Single Binary
- Build statik (CGO off bila memungkinkan) untuk x86_64 & arm64.
- Install location:
  - user: `~/.local/bin/talpa`
  - system: `/usr/local/bin/talpa`

### 11.2 Installer Script
- `curl -fsSL ... | bash` dengan opsi:
  - `-s latest` (stable release)
  - `-s <version>`
  - `--prefix` custom
- Verifikasi checksum.

### 11.3 Paket Distro (v1.x)
- `.deb`, `.rpm`, AUR PKGBUILD
- Flatpak/Snap **untuk CLI** optional (tergantung kebutuhan distribusi)

---

## 12) Compatibility Matrix (Target)
| Area | Target Minimum |
|---|---|
| Kernel | 5.x (best effort) |
| Distro | Ubuntu/Debian, Fedora/RHEL-family, Arch, openSUSE |
| Init | systemd (utama), non-systemd best-effort (tanpa optimize tertentu) |
| Desktop | GNOME/KDE/Wayland/X11 (analyze/status mostly independent) |
| Arch | amd64, arm64 |

---

## 13) Technical Architecture (High-level)
- Language: **Go**
- CLI framework: cobra / urfave/cli (pilih salah satu)
- TUI: Bubble Tea + Lip Gloss (Mole memakai ekosistem ini; cocok untuk UX konsisten)
- Metrics: gopsutil + /proc parsing fallback
- Scanner:
  - concurrent directory walk
  - size aggregation + age buckets
  - max depth + excludes
  - symlink loop protection
- Rules:
  - built-in rule sets (Go structs) untuk MVP
  - opsi ekstensi via YAML/JSON untuk v1.x

---

## 14) QA & Test Plan
- Unit test: path validator, rule matcher, symlink safety, recent-project detector
- Integration test (containerized):
  - simulated home dir dengan fixture file
  - distro matrix CI (ubuntu/fedora/arch containers) untuk command compatibility
- “Golden tests” untuk output `--json`
- Manual test checklist untuk setiap command

---

## 15) Success Metrics
- Space reclaimed per run (median)
- Time-to-scan (p50/p95)
- Crash-free sessions
- False positive rate (jumlah laporan “file penting terhapus” → target ~0)
- Adoption: stars/issues/PR community (opsional)

---

## 16) Open Questions
1. Apakah `talpa optimize` perlu modul per Desktop Environment (GNOME/KDE) atau cukup minimal cross-distro?
2. Apakah default purge mencakup docker (images/volumes) atau dipisah sebagai command `talpa docker`?
3. Kebijakan “trash by default” untuk `clean` (user-level) vs permanent delete (lebih cepat). MVP: permanent delete untuk clean (dengan confirm), trash untuk analyze.

---

## 17) Appendix — Mapping dari Mole → Talpa
| Mole | Talpa (Linux) | Catatan |
|---|---|---|
| clean | clean | kategori disesuaikan XDG + distro cache |
| uninstall | uninstall | dukung apt/dnf/pacman + snap/flatpak |
| optimize | optimize | aksi aman & modular |
| analyze | analyze | exclude /proc /sys /dev /run |
| status | status | gopsutil + /proc |
| purge | purge | scan dirs configurable |
| installer | installer | .deb/.rpm/.AppImage dsb |
| touchid | auth (opsional) | Linux: polkit/doas/sudo caching |
| completion/update/remove | completion/update/remove | paritas |

