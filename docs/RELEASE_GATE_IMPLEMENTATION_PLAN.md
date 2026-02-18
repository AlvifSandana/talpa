# Implementation Plan: Gate Rilis Talpa CLI

Dokumen ini berisi rencana implementasi berbasis checklist gate rilis untuk mempercepat keputusan **GO rilis** dengan pendekatan **minimal-diff**.

## Tujuan

Menutup gap pada area:

1. **Product**: deep cleanup system-level pada `clean`, flow interaktif `analyze`, dan baseline metric `status`.
2. **Safety**: double-confirm untuk aksi high-risk lintas command, serta perilaku whitelist konsisten dengan dokumentasi.
3. **Quality/Contract**: sinkronisasi `docs/JSON_SCHEMA.md` dengan output aktual.
4. **Release Readiness**: kesesuaian klaim docs dengan implementasi dan pipeline release end-to-end (artifact/checksum/publish).

---

## Sprint Plan (1–2 Minggu)

## Sprint 1 — Blocker Rilis (High)

### H1 — Safety: Double-confirm high-risk lintas command

- **Tujuan**: Wajib 2 lapis konfirmasi untuk aksi berisiko tinggi.
- **Area/File (perkiraan)**:
  - `cmd/root.go`
  - `internal/app/common/guards.go`
  - `internal/app/common/guards_test.go`
  - `internal/app/remove/service.go`
  - `internal/app/uninstall/service.go`
  - `internal/app/installer/service.go`
  - `internal/app/optimize/service.go`
  - `internal/app/clean/service.go`
  - `internal/app/purge/service.go`
  - `scripts/ci/smoke_negative.sh`
  - `docs/SAFETY_MODEL.md`
- **Dependensi**: None
- **Risiko**: Medium (potensi mengubah scripted flow lama)
- **Effort**: M
- **Acceptance Criteria**:
  - Aksi high-risk gagal tanpa konfirmasi lapis-2.
  - Error message konfirmasi konsisten lintas command.
  - Smoke negative memverifikasi minimal 3 command high-risk.

### H2 — Product: `clean` system-level opt-in

- **Tujuan**: Menambahkan deep cleanup system-level secara opt-in dan aman.
- **Area/File (perkiraan)**:
  - `cmd/clean.go`
  - `internal/domain/rules/rules.go`
  - `internal/app/clean/service.go`
  - `internal/domain/safety/validator.go`
  - `internal/domain/safety/validator_test.go`
  - `internal/app/clean/service_test.go`
  - `internal/app/clean/service_golden_test.go`
  - `internal/app/clean/testdata/clean_dry_run.golden.json`
  - `docs/RULESET_REFERENCE.md`
  - `docs/PRD.md`
  - `docs/SAFETY_MODEL.md`
- **Dependensi**: H1
- **Risiko**: High (scope root path/safety)
- **Effort**: L
- **Acceptance Criteria**:
  - `clean --system --dry-run --json` menampilkan kandidat system-level.
  - Non-root apply menghasilkan `skipped` + reason.
  - Validator/safety test tetap lulus.

### H3 — Product: Baseline metric `status` sesuai PRD

- **Tujuan**: Memenuhi baseline metric status yang disepakati untuk rilis.
- **Area/File (perkiraan)**:
  - `internal/app/status/service.go`
  - `internal/infra/system/processes.go`
  - `internal/app/status/service_golden_test.go`
  - `internal/app/status/testdata/status_dry_run.golden.json`
  - `docs/JSON_SCHEMA.md`
  - `docs/PRD.md`
- **Dependensi**: None (bisa paralel dengan H2)
- **Risiko**: Medium (variabilitas procfs Linux)
- **Effort**: M
- **Acceptance Criteria**:
  - Output JSON `status` memuat field baseline final.
  - Golden test stabil.
  - Smoke JSON status lulus.

### H4 — Quality/Contract: Sinkronisasi JSON schema 100%

- **Tujuan**: Menghilangkan mismatch docs vs output aktual command.
- **Area/File (perkiraan)**:
  - `docs/JSON_SCHEMA.md`
  - `scripts/ci/check_json_schema_sync.sh` (baru)
  - `.github/workflows/ci.yml`
  - `scripts/ci/run_local_ci.sh`
  - `internal/app/**/testdata/*.golden.json` (jika output berubah)
- **Dependensi**: H2, H3
- **Risiko**: Low
- **Effort**: S
- **Acceptance Criteria**:
  - CI fail bila schema docs tidak sinkron.
  - Semua command `--json` tercakup checker.

---

## Sprint 2 — Completeness + Release Readiness

### H5 — Product: Analyze action flow interaktif end-to-end

- **Tujuan**: Menyediakan alur interaktif analyze dari inspect hingga eksekusi aksi aman.
- **Area/File (perkiraan)**:
  - `cmd/analyze.go`
  - `cmd/interactive.go`
  - `internal/app/analyze/service.go`
  - `internal/app/common/guards.go`
  - `cmd/interactive_test.go`
  - `internal/app/analyze/service_test.go`
  - `internal/app/analyze/service_golden_test.go`
  - `docs/PRD.md`
  - `docs/SAFETY_MODEL.md`
  - `docs/JSON_SCHEMA.md`
- **Dependensi**: H1
- **Risiko**: Medium-High (UX + safety)
- **Effort**: L
- **Acceptance Criteria**:
  - Flow analyze action berjalan dari mode interaktif.
  - Permanent delete selalu mewajibkan double-confirm.
  - `--dry-run` tidak memodifikasi file.

### M1 — Safety: Whitelist behavior sesuai docs

- **Tujuan**: Menyamakan perilaku whitelist di implementasi dan dokumentasi.
- **Area/File (perkiraan)**:
  - `internal/infra/config/store.go`
  - `internal/domain/safety/validator.go`
  - `internal/domain/safety/validator_test.go`
  - `docs/SAFETY_MODEL.md`
  - `docs/CONTRIBUTING.md`
- **Dependensi**: H1
- **Risiko**: Medium (false allow / false block)
- **Effort**: M
- **Acceptance Criteria**:
  - Test whitelist minimal 5 skenario lulus.
  - Contoh di docs sesuai perilaku aktual.

### H6 — Release readiness end-to-end

- **Tujuan**: Menyiapkan flow rilis end-to-end (dry-run) dan menyamakan klaim docs.
- **Area/File (perkiraan)**:
  - `.github/workflows/release.yml` (baru)
  - `scripts/ci/release_dry_run.sh` (baru)
  - `docs/RELEASE_PROCESS.md`
  - `README.md`
  - `docs/PRD.md`
- **Dependensi**: H4 + stabilisasi fitur prioritas
- **Risiko**: Medium
- **Effort**: M
- **Acceptance Criteria**:
  - Dry-run menghasilkan artifact linux amd64/arm64 + SHA256.
  - Verifikasi checksum otomatis lulus.
  - Tidak ada klaim docs yang over-claim terhadap implementasi.

---

## Urutan Eksekusi & Paralelisasi

1. **H1** (wajib paling awal)
2. **H2 + H3** (paralel)
3. **H4** (setelah H2/H3 stabil)
4. **H5 + M1** (paralel terbatas)
5. **H6**

> Catatan: H6 boleh mulai scaffold workflow lebih awal, tetapi finalisasi menunggu output contract stabil (H4).

---

## Definisi Done (GO Rilis)

Rilis dinyatakan **GO** jika semua kondisi berikut terpenuhi:

- Product gate:
  - `clean` system-level opt-in aman dan teruji.
  - `analyze` action flow interaktif end-to-end tersedia.
  - `status` memenuhi baseline metric PRD final.
- Safety gate:
  - Double-confirm high-risk konsisten lintas command.
  - Whitelist behavior sesuai docs + test memadai.
- Quality/Contract gate:
  - `docs/JSON_SCHEMA.md` sinkron 100% dengan output command (dijaga CI).
- Release readiness gate:
  - Kesesuaian klaim docs vs implementasi.
  - Release dry-run artifact + checksum + publish flow lulus.
- CI gate:
  - Semua command verifikasi wajib lulus tanpa flaky.

---

## Command Verifikasi Wajib (Sebelum GO)

### Core

```bash
go test ./...
go vet ./...
go test -race ./...
bash scripts/ci/smoke_json.sh
bash scripts/ci/smoke_negative.sh
bash scripts/ci/run_local_ci.sh
```

### Contract / Schema

```bash
bash scripts/ci/check_json_schema_sync.sh
```

### Release dry-run

```bash
bash scripts/ci/release_dry_run.sh
```

### Build artifact + checksum sanity (manual fallback)

```bash
mkdir -p dist
GOOS=linux GOARCH=amd64 go build -o dist/talpa-linux-amd64 ./cmd/talpa
GOOS=linux GOARCH=arm64 go build -o dist/talpa-linux-arm64 ./cmd/talpa
sha256sum dist/talpa-linux-amd64 dist/talpa-linux-arm64 > dist/SHA256SUMS
sha256sum -c dist/SHA256SUMS
```
