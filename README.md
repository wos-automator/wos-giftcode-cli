# WOS Gift-Code Redemption Client

A small command-line client for redeeming gift codes in
*Whiteout Survival* (WOS). It talks to the official
`wos-giftcode-api.centurygame.com` API and automates the flow:
identify player -> fetch captcha -> solve captcha -> redeem code.

There are two implementations:

- **Node.js** - inside the `js/` folder (`js/wos-client.js`)
- **Go** - inside the `go/` folder (`go/main.go` / `go/go.mod`)

> **Note:** This project is reverse-engineered and intended for learning and
> automation purposes only. Use at your own risk and in compliance with the
> game's terms of service.

## Requirements

- **Node.js >= 18** (for the Node version; uses built-in `fetch`)
- **Go >= 1.21** (for the Go version)
- An OS with a default image viewer/browser
  (macOS `open`, Windows `start`, Linux `xdg-open`)

## Installation

No external dependencies are required. Both versions use only standard
libraries (Node.js: `crypto`, `fs`, `os`, `path`, `readline`,
`child_process`; Go: `crypto/md5`, `net/http`, `encoding/*`, etc.).

```bash
git clone <repo-url>
cd wos-giftcode-cli
```

## Usage

### Node.js

Run directly with `node`:
```bash
node js/wos-client.js [fid] [giftcode]
```
Or run using `npm` from the `js/` folder:
```bash
cd js
npm start [fid] [giftcode]
```

### Go

Run directly:
```bash
go run ./go [-u userid] <giftcode>
```
Or build a binary and run it:
```bash
go build -o wos-client ./go && ./wos-client [-u userid] <giftcode>
```

Arguments:

- **Go:** `-u userid` (optional) and `<giftcode>` (required)
- **Node.js:** `[fid] [giftcode]` (legacy behavior)

If `-u` is not provided, the user ID (`fid`) is resolved in this order:
CLI flag `-u` > environment variable `DEFAULT_FID` > local `.env` > global `~/.wos-cli-env` > built-in default `123456789`.

You can create a global configuration file in your home directory (`~/.wos-cli-env`) or a local project `.env` file containing:

```bash
DEFAULT_FID=123456789
```

### Examples

Default ID from ENV/.env/default:

```bash
go run ./go HAPPY2026
```

Explicit user ID and code:

```bash
node js/wos-client.js 123456789 HAPPY2026
go run ./go -u 123456789 HAPPY2026
```

### Build Script (Go)

Use `go/build.sh` for reproducible builds:

```bash
./go/build.sh          # Build binary for current platform (-> dist/)
./go/build.sh all      # Build binaries for macOS/Linux/Windows (amd64 & arm64)
./go/build.sh clean    # Remove dist/
```

Binaries are written to `dist/` in the repository root. If `go` is not in `PATH`, the script checks
common install locations; you can also run with `GO=/path/to/go ./go/build.sh`.

## Flow

1. **Identify player** - Calls `/player` and prints player data.
2. **Fetch captcha** - Calls `/captcha`; image is returned as Base64 data URI.
3. **Show captcha** - Decodes image, writes it to a temporary file, and opens it
   in the default viewer/browser.
4. **Enter captcha** - Reads solved captcha text from terminal input.
5. **Redeem code** - Calls `/gift_code` and prints the result.

## Technical Notes

- All requests are signed via `appendSign()`: parameters are sorted
  alphabetically, URL-encoded, concatenated with a fixed `SALT`, and hashed
  with MD5.
- Requests are sent as `application/x-www-form-urlencoded`.

## CI/CD (GitHub Actions)

There are two workflows in `.github/workflows/`.

### 1. Build (`build.yml`)

Runs on push to `main`/`master`, pull requests, tags `v*`, and manual dispatch
(`workflow_dispatch`). It runs `go vet` inside the `go/` folder, builds all targets with
`go/build.sh all`, and uploads `dist/` as build artifact (`wos-client-dist`).

### 2. Release (`release.yml`)

Runs only when pushing a `v*` tag and creates a GitHub release:

1. **Signature verification (SSH):** tag must be SSH-signed. The workflow writes
   allowed signers to `allowed_signers` and verifies with `git tag -v`.
2. **Build** all platforms via `go/build.sh all`.
3. **Changelog:** generates release notes from commits since the previous tag.
4. **Assets:** attaches Go binaries (`dist/*`), JS version (`js/wos-client.js`),
   `README.md`, and API docs (`BACKEND_PROTOCOL.md`).
5. **Release** is published with generated notes.

#### Release Prerequisites

- **Create signed tags** (SSH signing):

  ```bash
  # One-time local Git config:
  git config gpg.format ssh
  git config user.signingkey ~/.ssh/id_ed25519.pub

  git tag -s v1.0.0 -m "Release v1.0.0"
  git push origin v1.0.0
  ```

- **Set repository secret `SSH_ALLOWED_SIGNERS`**. It must contain one line in
  OpenSSH `allowed_signers` format:

  ```text
  your-email@example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
  ```

  Use the same public key used for tag signing, and an email that matches the
  tagger identity.

## License

This project is licensed under the **GNU General Public License v3.0**.
See [`LICENSE`](./LICENSE) for the full text.

