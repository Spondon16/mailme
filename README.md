# mailme

> Fast, secure, and powerful disposable temporary email CLI written in Go, powered by [mail.tm](https://mail.tm/) with optional support for additional providers.

`mailme` allows you to generate, read, watch, and manage temporary inboxes directly from your terminal.

---

## Features

- **Instant Address Generation**: Create custom or randomly generated disposable email addresses.
- **Cross-Platform Clipboard**: Automatically copies generated emails to clipboard (supports macOS `pbcopy`, Linux Wayland `wl-copy`, Linux X11 `xclip`/`xsel`, and WSL/Windows `clip.exe`).
- **Real-time SSE Watching**: Listen for incoming emails in real-time via Mercure Server-Sent Events (SSE) with graceful fallback to interval polling.
- **Full Message & `.eml` Support**: Read formatted headers/body, view raw text, view raw HTML, or inspect and download full raw RFC 822 `.eml` source messages.
- **Attachment Downloads**: Fetch and save attachments safely with atomic disk writes.
- **Inbox Pagination**: Browse paginated inboxes beyond the first 30 messages (`--page`).
- **Read Receipt Synchronization**: Automatically marks messages as read/seen on the server upon viewing.
- **Multi-Account Switching**: Store multiple temporary inboxes and effortlessly switch between active accounts.
- **Resilient & Atomic Storage**: State is saved with atomic writes and strict permissions (`0700` dir / `0600` file). Automatic token refresh handles expired JWTs.

---

## Installation

### Prebuilt Binaries

Download the binary for your platform from the [Releases page](https://github.com/Spondon16/mailme/releases/latest), then:

```sh
# Linux / macOS — make it executable and move it onto your PATH
chmod +x mailme-<platform>-<arch>
sudo mv mailme-<platform>-<arch> /usr/local/bin/mailme
```

Each release includes a `SHA256SUMS.txt` — verify a download with `sha256sum -c SHA256SUMS.txt` before running it.

### From Source (Requires Go 1.22+)

```sh
# Clone repository
git clone https://github.com/Spondon16/mailme.git
cd mailme

# Build binary
go build -o mailme .

# Optional: Install to your PATH
sudo install -m755 mailme /usr/local/bin/mailme
```

### Cross-Compilation

```sh
# Linux (amd64 / arm64)
GOOS=linux GOARCH=amd64 go build -o dist/mailme-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o dist/mailme-linux-arm64 .

# macOS (Apple Silicon / Intel)
GOOS=darwin GOARCH=arm64 go build -o dist/mailme-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o dist/mailme-darwin-amd64 .

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o dist/mailme-windows-amd64.exe .
```

---

## Quick Start

```sh
# 1. Generate a new email (automatically copied to clipboard)
mailme generate

# 2. Check inbox
mailme inbox

# 3. Read an email by ID
mailme read <message-id>

# 4. Stream incoming messages in real-time
mailme watch

# 5. Delete active account
mailme delete account
```

---

## Command Reference

### Generate Address (`generate`, `g`)
```sh
# Random address
mailme generate

# Custom username with random domain
mailme generate myuser

# Custom username with flag
mailme generate -u myuser

# Specific full address and custom password
mailme generate myuser@example.com -w MySecretPass123!

# Specific domain
mailme generate -d example.com
```

### Inbox (`inbox`, `m`)
```sh
# View active inbox
mailme inbox

# View specific account's inbox
mailme inbox user@example.com

# Pagination (page 2, 3, etc.)
mailme inbox --page 2

# Output as JSON (ideal for automation and scripts)
mailme inbox --json
```

### Read Message (`read`)
```sh
# Read formatted message and headers (auto-marks as read)
mailme read <message-id>

# View raw plain text only
mailme read <message-id> --raw

# View raw HTML body
mailme read <message-id> --html

# View raw RFC 822 / EML source
mailme read <message-id> --eml
```

### Real-Time Watcher (`watch`)
```sh
# Watch active inbox in real-time via Mercure SSE
mailme watch

# Force polling instead of SSE with custom interval
mailme watch --poll -i 3

# Check once and exit
mailme watch --once
```

### Download (`download`)
```sh
# Download an attachment
mailme download <message-id> <attachment-id> -o ~/Downloads

# Download entire raw message as .eml file
mailme download <message-id> --eml -o ~/Downloads/email.eml
```

### Account Management (`accounts`, `me`, `d`)
```sh
# List all saved accounts
mailme accounts

# Show active account details
mailme me
# or
mailme accounts show

# Switch active account
mailme accounts use user@example.com

# Delete current account (remote + local)
mailme delete account
# or shortcut
mailme d -f

# Delete specific message
mailme delete message <message-id>
```

### Domains (`domains`)
```sh
# List all currently available domains
mailme domains
```

---

## Additional Providers

`mailme` defaults to [mail.tm](https://mail.tm/), but also supports a few extra providers via [TempMail-UnofficialAPI](https://github.com/josskixg/TempMail-UnofficialAPI):

```sh
mailme generate -p tempmail.plus
mailme generate -p tempmailc
mailme generate -p mailnesia
```

These providers assign a full random address automatically — `-u`/`-d` aren't supported for them. They also don't support `--eml`/attachment download, per-message delete, or `watch`'s real-time SSE stream (it transparently falls back to polling); `mailme domains` isn't meaningful for them either.

**Why not guerrillamail, tempmail.lol, or dropmail?** They were evaluated too, but each requires a server-issued session token to check the inbox later. The wrapper library only ever holds that token in an unexported struct field with no way to read it back out, so there's no way to persist it to `accounts.json` — `generate` would work, then `inbox`/`read` would silently fail on the very next command, since `mailme` is a fresh process each time. They're excluded until upstream exposes a way to save and restore that session state. `tempmail.plus`, `tempmailc`, and `mailnesia` were chosen specifically because their inbox lookups are plain HTTP calls keyed by the email address itself — no session required.

---

## Configuration & Security

- Accounts and JWT tokens are stored in `os.UserConfigDir()/mailme/accounts.json` (`~/.config/mailme/` on Linux, `%APPDATA%\mailme\` on Windows, `~/Library/Application Support/mailme/` on macOS).
- Directory permissions are created with `0700` and files with `0600`.
- Updates to configuration and downloaded files use atomic writes (`.tmp` + `os.Rename`) to prevent data corruption.
- Passwords and tokens are stored locally on your machine to support seamless token auto-refresh upon expiration.
- **Trust model note:** passwords are stored in **plaintext**, not hashed or encrypted at rest — this is a deliberate tradeoff to support automatic re-authentication, not an oversight. This is low-risk for disposable/throwaway inboxes (mail.tm accounts have no recovery flow anyway), but do not reuse a real password when generating an address, and be aware that anyone with read access to your user account can read `accounts.json`.

---

## Development

```sh
go build ./...   # Compile all packages
go vet ./...     # Static analysis & vetting
gofmt -w .       # Format all files
```

---

## Credits

- [Mailsy](https://github.com/BalliAsghar/Mailsy) and [mailtm](https://github.com/ABGEO/mailtm) — feature and design inspiration.
- [TempMail-UnofficialAPI](https://github.com/josskixg/TempMail-UnofficialAPI) — backs the additional providers above.
