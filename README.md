# SProject

A cross-platform Remote Access Tool with traffic encryption, remote file system management, screenshot capture, and persistence mechanism. Written in Go from scratch without using third-party RAT frameworks.

**Stack:** Go 1.26 · TLS (ECDSA P-256, self-signed) · kbinani/screenshot · golang.org/x/sys


---

## Features

- Encrypted connection (TLS) — traffic is protected at the transport layer without the need for manual certificate configuration
- Managing multiple clients simultaneously through an interactive CLI
- Remote shell command execution (`cmd /c` on Windows, `sh -c` on Unix)
- Bidirectional file transfer (upload / download)
- Screenshot capture and retrieval to server
- Desktop wallpaper change (Windows, macOS, Linux GNOME)
- Text-to-Speech on the client side (Windows)
- Persistence on Windows: copying to `C:\ProgramData`, running via Task Scheduler with `HIGHEST` privilege level at user login
- Cross-compilation for Windows / Linux / macOS from a single Makefile

---

## Build

The server address is set at compile time via `-ldflags`; the binary does not require configuration files.

```bash
# Server (Linux amd64)
make build-server

# Clients for all platforms
make clients SERVER_ADDR=1.2.3.4:4444

# Deploy server to VPS + build all clients
make deploy VPS=root@1.2.3.4 SERVER_ADDR=1.2.3.4:4444
```

| Target                  | Result                                      |
|-------------------------|---------------------------------------------|
| `build-server`          | `server_linux` (CGO_ENABLED=0)              |
| `build-client-linux`    | `client_linux`                              |
| `build-client-windows`  | `client.exe` (no console window, `-H=windowsgui`) |
| `build-client-mac`      | `client_mac`                                |
| `clients`               | All three clients at once                   |
| `deploy`                | Build + scp server to VPS                   |

---

## Usage

**Running the server:**
```bash
./server_linux
```
The server listens on port `4444`. When a new client connects, a notification is printed to the console.

**Operator CLI — without an active client:**

| Command   | Description                     |
|-----------|------------------------------|
| `list`    | List of connected clients |
| `use <N>` | Select a client by number    |
| `exit`    | Shut down the server     |

**Operator CLI — with an active client:**

| Command                 | Description                                           |
|-------------------------|----------------------------------------------------|
| `info`                  | Username, hostname, MAC address, system time     |
| `ls [path]`             | Directory contents                              |
| `cd <path>`             | Change working directory                         |
| `download <path>`       | Download a file from the client                             |
| `upload <file>`         | Upload a file to the client                           |
| `screenshot` / `screen` | Take a screenshot and save to server              |
| `wallpaper <file>`      | Set wallpaper (Windows / macOS / Linux GNOME)    |
| `speak <text>`          | Reproduce text via TTS                      |
| `<command>`             | Execute an arbitrary shell command               |
| `q`                     | Return to client selection                         |

---

## Project Structure

```
cmd/
  server/          — server entry point
  client/          — client entry point (platform-dependent initialization)
internal/
  server/          — TLS listener, client pool, operator CLI
  client/          — connection, command dispatcher, persistence
  protocol/        — reliable message serialization, file transfer
  system/          — system information collection
```




---

## TLS and Threat Model

The server generates an ephemeral ECDSA P-256 certificate in memory on each run — no files, no CA. The client connects with `InsecureSkipVerify: true`, meaning it does not verify the certificate chain.

This is a deliberate trade-off, acceptable for this use case:

- **The purpose of TLS here is traffic encryption**, not server authentication. The connection is protected against passive interception (DPI, network sniffers, Windows Defender Network Inspection).
- **MITM is not in the threat model.** The client already knows the server address hardcoded at compile time. The infrastructure assumes there is no hostile intermediary between client and server — it's a managed environment of your own.
- **PKI infrastructure is redundant.** Creating a CA, signing certificates, and rotating them — significant operational overhead that provides no real security benefit in this deployment scheme (single server, client with hardcoded address).

If the threat model changes (public infrastructure, multiple operators) — the correct solution: certificate pinning on the client side with the certificate fingerprint hardcoded at compile time.

---

## Legal Notice

This tool is developed for educational purposes and for use exclusively on devices that you own or have explicit written permission to manage. Unauthorized access to other people's systems is a criminal offense in most jurisdictions. The author is not responsible for any use of this software for illegal purposes.
