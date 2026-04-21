# Self-Hosting SchemaPing

**Author:** Rubens Antonio Rosa  
**Date:** 2026-04-21  
**Version:** 0.1.0

---

## Table of Contents

- [Build the binary](#build-the-binary)
- [Linux — systemd](#linux--systemd)
- [macOS — launchd](#macos--launchd)
- [Running manually in background](#running-manually-in-background)

---

## Build the binary

```bash
git clone https://github.com/rubensantoniorosa2704/schemaping-worker.git
cd schemaping-worker
go build -o schemaping ./cmd/schemaping
```

Move the binary to a location in your `$PATH`:

```bash
sudo mv schemaping /usr/local/bin/schemaping
```

---

## Linux — systemd

Create a config file at `/etc/schemaping/config.yaml`:

```yaml
monitors:
  - name: payments-api
    url: https://api.example.com/v1/payments
    interval: 5m
    headers:
      Authorization: Bearer YOUR_TOKEN
```

Create the service file at `/etc/systemd/system/schemaping.service`:

```ini
[Unit]
Description=SchemaPing API drift monitor
After=network.target

[Service]
ExecStart=/usr/local/bin/schemaping run --config /etc/schemaping/config.yaml
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable schemaping
sudo systemctl start schemaping
```

Check logs:

```bash
journalctl -u schemaping -f
```

---

## macOS — launchd

Create a config file at `~/.schemaping/config.yaml`.

Create `~/Library/LaunchAgents/com.schemaping.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.schemaping</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/schemaping</string>
    <string>run</string>
    <string>--config</string>
    <string>/Users/YOUR_USER/.schemaping/config.yaml</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/tmp/schemaping.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/schemaping.log</string>
</dict>
</plist>
```

Load the agent:

```bash
launchctl load ~/Library/LaunchAgents/com.schemaping.plist
```

Check logs:

```bash
tail -f /tmp/schemaping.log
```

---

## Running manually in background

The simplest option — no service manager needed:

```bash
nohup schemaping run --config ./config.yaml >> ~/.schemaping/schemaping.log 2>&1 &
```

Check logs:

```bash
tail -f ~/.schemaping/schemaping.log
```

Stop it:

```bash
pkill schemaping
```
