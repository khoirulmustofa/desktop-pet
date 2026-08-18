# Desktop Pet (Windows)

Desktop pet kucing ringan untuk Windows, ditulis murni dalam Go + Win32 API.
Tanpa Electron, tanpa browser engine, tanpa WebView.

```
      🐱
     /|\        transparan, selalu di atas aplikasi lain,
      |         bergerak random, bisa di-drag,
                punya system tray, RAM < 15 MB, CPU ~0% saat idle
```

## Fitur

- Window borderless, transparan (per-pixel alpha), always-on-top, tanpa tombol taskbar.
- 5 animasi: `idle`, `walk`, `jump`, `sleep`, `happy` dengan state machine.
- Behavior engine: pet idle 5–15 detik lalu random (walk/jump/sleep/happy).
- Gerakan horizontal dengan delta-time + batas layar (work area monitor).
- Fisika lompat sederhana (gravity + jump velocity).
- **Multi-monitor**: drag pet ke monitor lain, atau klik kanan/tray → "Move to
  Monitor 1/2/…" (daftar monitor aktif, dengan centang pada monitor saat ini).
- Drag karakter dengan mouse; klik kanan / tray icon membuka menu.
  **Click-through** meneruskan klik kiri ke aplikasi di bawahnya, tapi klik
  kanan pada pet tetap membuka menu (jadi mode click-through selalu bisa
  dimatikan dari menu pet).
- System tray: Show/Hide, Pause/Resume, Always on Top, Click-through,
  Start with Windows, FPS & Scale, Exit.
- Konfigurasi `config.json` di `%APPDATA%\DesktopPet\`.
- Log penting di `%APPDATA%\DesktopPet\logs\app.log`.
- Semua sprite PNG di-embed ke dalam satu executable.

## Requirement

- Windows 10/11 (64-bit).
- Go 1.21+ untuk development (build menghasilkan exe mandiri, tanpa runtime).
- Tidak perlu GCC / cgo / dependency tambahan (zero dependency).

## Menjalankan Development

```powershell
# toolchain (contoh; sesuaikan dengan lokasi go.exe di mesin Anda)
$go = "C:\Program Files\FlyEnv-Data\app\static-go-1.26.6\bin\go.exe"

# regenerasi sprite (opsional, sudah ada di assets/pet)
& $go run ./cmd/genassets

# jalankan dengan console (debug, stderr tampil di terminal)
& $go run ./cmd/desktop-pet

# atau build debug exe
& $go build -o dist\desktop-pet.exe ./cmd/desktop-pet
```

Konfigurasi disimpan di `%APPDATA%\DesktopPet\config.json`. Jika corrupt,
file di-backup ke `config.bak.json` dan aplikasi lanjut dengan default.

## Build Release

```powershell
# PowerShell di root project
.\build.ps1            # release (GUI, tanpa console)
.\build.ps1 -Debug     # build debug (dengan console)
```

Hasil: `dist\desktop-pet.exe` — satu file, siap dipindah/di-share.
Icon executable (.ico) belum disematkan di MVP (membutuhkan `rsrc`); hal ini
dapat ditambahkan di phase berikutnya.

## Menambahkan Sprite

Struktur folder:

```text
assets/pet/
├── idle/   01.png, 02.png, ...
├── walk/   01.png, 02.png, ...
├── jump/   ...
├── sleep/  ...
└── happy/  ...
```

- Ukuran frame bebas; semua frame satu animasi harus sama ukurannya (mis. 96×96).
- PNG dengan alpha channel (transparan). Hanya pixel karakter yang ditampilkan.
- Penamaan diurutkan abjad (01, 02, … 09, 10).
- Setelah menambahkan/mengubah PNG, jalankan `go run ./cmd/genassets`? Tidak perlu —
  genassets hanya menghasilkan sprite bawaan. Ganti file di `assets/pet/<nama>/`
  lalu jalankan `.\build.ps1` (file di-embed ke exe saat build).

Sprite bawaan digambar secara prosedural oleh `internal/spritegen` (dipakai juga
sebagai fallback runtime bila asset embed rusak). Untuk memakai karakter sendiri,
cukup ganti PNG di `assets/pet/`.

## Menambahkan Animasi

1. Buat folder `assets/pet/<nama>/` berisi frame PNG.
2. Muat di `internal/app/app.go` → `loadAnimations()`.
3. Daftarkan state baru di `internal/animation/state.go` (enum `State`).
4. Tambahkan mapping di map `anims` di `loadAnimations()`.
5. Atur perilakunya di behavior engine (lihat di bawah).

Format animasi (lihat `internal/animation/animation.go`):

```go
type Animation struct {
    Name   string
    Frames []*image.RGBA
    FPS    int
    Loop   bool // false = play sekali lalu diam di frame terakhir
}
```

## Membuat Behavior Baru

Behavior diatur oleh:

- `internal/behavior/behavior.go` — enum `Action` dan `Weights` (probabilitas).
- `internal/app/app.go` → `planNext()` — keputusan state berikutnya setelah idle.
- `internal/app/app.go` → `tick()` — loop utama tiap frame (switch per state).

Menambah aksi baru: tambahkan konstanta di `Action`, bobot di `Weights`, kasus di
`planNext()` dan `tick()`. Semua berjalan di satu goroutine (UI thread) sehingga
tidak ada masalah konkurensi.

## Arsitektur

```text
cmd/desktop-pet     entry point (main.go)
cmd/genassets       generator sprite PNG bawaan
internal/app        orkestrasi, behavior, movement, physics (UI thread)
internal/window     window Win32 + semua binding syscall (win32.go)
internal/renderer   load PNG + blit ke layered window (UpdateLayeredWindow)
internal/animation  state machine + frame player
internal/behavior   random behavior engine
internal/movement   gerakan delta-time + boundary
internal/tray       Shell_NotifyIcon + popup menu
internal/config     config.json + auto-start registry (HKCU Run)
internal/monitor    enumerasi & work area monitor (multi-monitor)
internal/spritegen  penggambar kucing prosedural
```

Desain utama:

- Satu OS thread (locked) untuk window + GDI + message loop.
- `SetTimer` 30/60 FPS; render hanya ketika frame berubah / pet bergerak.
- Tidak ada goroutine per-frame, tidak ada busy loop.
- Zero dependency; semua Win32 via `syscall.NewLazyDLL`.

## Multi-monitor

- Enumerasi monitor via `EnumDisplayMonitors` (`internal/monitor/monitor.go`).
- Pet menempel pada monitor tempat ia berada: `groundY` & batas gerak mengikuti
  work area monitor tersebut (`bindMonitor` di `internal/app/app.go`).
- Pindah monitor lewat:
  - **Drag** — tarik pet ke monitor lain; saat dilepas ia otomatis menempel ke
    monitor tersebut.
  - **Menu** — klik kanan pet / tray icon → `Move to Monitor 1/2/…` (hanya
    tampil bila ada >1 monitor aktif).
- Menampilkan "Move to Monitor N" berurutan kiri→kanan sesuai posisi monitor;
  yang primary diberi label `(Primary)`.
- Safeguard: jika monitor dicabut, pet otomatis menempel ke monitor terdekat.

## Performance

Target & hasil pengukuran (dev machine):

| Metrik   | Target | Hasil    |
|----------|--------|----------|
| RAM      | < 30 MB| ~13.6 MB |
| CPU idle | < 1%   | ~0.16%   |
| Startup  | < 1 s  | < 1 s    |
| FPS      | 30–60  | 30 (configurable) |

## Testing

```powershell
& $go test ./...
```

Unit test mencakup animation, movement (boundary), behavior (weights), config
(corrupt fallback), dan renderer (frame alpha).

## Release (Cara Buat Release)

1. Jalankan `.\build.ps1`.
2. Salin `dist\desktop-pet.exe` ke lokasi target (bisa tanpa folder lain).
3. (Opsional) Aktifkan "Start with Windows" lewat menu tray.

## Roadmap / Phase 2

- Multiple pet, multiple monitor, suara, pet selection, configuration UI,
  dan integrasi coding companion (VS Code build success → Happy, dsb.)
  sebagai module terpisah.
