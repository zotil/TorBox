# TorBox ChatSecure (Go)

Encrypted web chat room to communicate in group or one-to-one over an Onion Service v3. All messages feature End-to-End Encryption (E2EE) using OpenPGP keys managed entirely on the client side in the browser.

Implemented in Go as a high-performance, single standalone binary (`tcs`) with zero external runtime dependencies and anti-forensic ephemeral RAM storage.

## Architecture & Features

1. **Hybrid Resilient Model for Tor:**
   - **Send Channel:** Transactional `POST /api/messages/send` with client outbox queue and ACK.
   - **Receive Channel:** Hardened real-time WebSocket (`/ws` and `/socket.io/` compatibility) with Tor keepalive intervals (Ping: 60s, Pong timeout: 120s).
   - **Fallback Sync:** Cursor-based message synchronization (`GET /api/messages/sync?since_id=...&group_id=default`) for transparent recovery across network disconnects.

2. **SD Anti-Forensics (Zero Physical Storage Footprint):**
   - The SQLite database resides exclusively in RAM (`tmpfs` in `/dev/shm/tcs_<NAME>.sqlite3`).
   - If power is cut or the process shuts down, all messages and session keys disappear physically from memory with zero trace on the SD card.

3. **High Performance & Low Resource Usage:**
   - Single static executable replacing Python, Eventlet, Gunicorn, Peewee, and Flask-SocketIO.
   - Memory footprint reduced from ~150 MB to < 15 MB.

## Building

To build `tcs`, use the included build script:
```bash
./build.sh          # Builds tcs for current host OS/ARCH
./build.sh arm64    # Cross-compiles for Linux ARM64 (Raspberry Pi 3/4/5)
./build.sh armv6    # Cross-compiles for Linux ARMv6 (Raspberry Pi Zero/1)
./build.sh all      # Builds all target binaries
```

## Usage

### Production (TorBox)

TorBox scripts (`menu-onion`, `bin/start_tcs`) launch `tcs` directly:

```bash
./tcs -n <SERVICE_NAME> -od <ONION_DOMAIN>
```

When started:
- PID file is created in `pid/<SERVICE_NAME>.pid`
- UNIX domain socket listener is created in `/var/run/tcs_<SERVICE_NAME>.sock` (permissions `0777`)
- Ephemeral database is initialized in `/dev/shm/tcs_<SERVICE_NAME>.sqlite3`

### Development

To run locally on port 5000:
```bash
./tcs -n dev -od localhost --dev --debug
```

Visit `http://localhost:5000` to interact with the web interface.

### Stopping TCS

To stop `tcs`, send `SIGTERM` or `SIGINT`:
```bash
kill `cat pid/<SERVICE_NAME>.pid`
```
or
```bash
pkill -F pid/<SERVICE_NAME>.pid
```
This triggers an atomic graceful cleanup that deletes the UNIX socket, PID file, and ephemeral SQLite database in RAM.

### Command Line Options

```
Usage of ./tcs:
  -n, --name string
        Onion Service Name
  -od, --onion-domain string
        Onion domain where TCS will be published
  --dev
        Run in development mode (Default: 0/false)
  --debug
        Run in debug mode (Default: 0/false)
```
