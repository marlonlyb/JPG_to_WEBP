# JPG to WEBP

Offline desktop JPEG to WebP converter built with Wails, Go, and React.

## Using the converter

- Input comes from `Choose JPEG`, which opens the picker flow for exactly one local `.jpg` / `.jpeg` file.
- The picker keeps the WSL-friendly fallback chain and remembers the last folder when the native dialog can resolve one.

## Linux / WSL development

This repository is configured for a Linux Wails target, but `wails dev` / `wails build` only work when the local Linux toolchain is installed inside WSL.

### 1. Install the Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`, for example:

```bash
export PATH="$HOME/go/bin:$PATH"
```

If you do not want to change your shell config yet, you can use the helper wrapper in this repo instead:

```bash
./scripts/wails-wsl.sh doctor
```

### 2. Install Linux system dependencies

Ubuntu 24.04 / Debian example:

```bash
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

If your distro still provides `libwebkit2gtk-4.0-dev`, that also works.

When only `libwebkit2gtk-4.1-dev` is installed, run Wails with the `webkit2_41` build tag:

```bash
wails dev -tags webkit2_41
wails build -tags webkit2_41
```

The helper wrapper adds that tag automatically when it detects WebKitGTK 4.1 without 4.0:

```bash
./scripts/wails-wsl.sh dev
./scripts/wails-wsl.sh build
```

### 3. Run the repository preflight

```bash
./scripts/check-linux-wsl-prereqs.sh
```

The script checks:

- Go / npm / gcc / pkg-config / Wails CLI
- GTK3 development headers
- WebKitGTK 4.0 or 4.1 development headers
- WSL GUI bridge variables (`DISPLAY` / `WAYLAND_DISPLAY`)

### 4. Validate app builds

```bash
cd frontend && npm run build
go test ./backend/...
./scripts/wails-wsl.sh doctor
./scripts/wails-wsl.sh dev
```

If you prefer calling Wails directly, add `-tags webkit2_41` on WebKitGTK 4.1 systems.
