# cec-controller

CEC Controller is a Linux CLI application that listens for HDMI-CEC key events and translates them to Linux virtual
keyboard actions. It also reacts to system power events (startup, shutdown, sleep, resume) and can be configured for
custom key mappings.

> [!WARNING]
> **This project requires libcec-compatible hardware.** You need a physical HDMI-CEC adapter recognised by
> [libcec](https://libcec.pulse-eight.com/) — the most common option is the
> [Pulse-Eight USB–CEC Adapter](https://www.pulse-eight.com/p/104/usb-hdmi-cec-adapter). Some Raspberry Pi models
> expose a built-in CEC interface (`/dev/ttyAMA0`). Without compatible hardware the daemon will fail to open the
> adapter and exit immediately.

## Features

- **HDMI-CEC key listening:** Uses [libcec](https://libcec.pulse-eight.com/) via Go bindings to receive remote key
  presses from connected HDMI devices.
- **Virtual keyboard emulation:** Maps CEC keys to Linux key codes and triggers key events
  using [micmonay/keybd_event](https://github.com/micmonay/keybd_event). (See default key map [here](keymap.go))
- **Customizable key mapping:** Override or extend the CEC→Linux key map via CLI flags.
- **Power event hooks:** Responds to system startup, shutdown, sleep, and resume of the host machine and transmits
  corresponding CEC commands (e.g. "Power On", "Standby") to connected devices.
- **Active source switching:** Optionally claims the active HDMI source on startup so the TV switches input to this device automatically.
- **Shutdown protection:** Holds a systemd-logind delay inhibitor lock while sending CEC standby commands, ensuring the system waits for CEC to complete before sleeping or shutting down.
- **Systemd-ready:** Includes a sample systemd service file for robust startup and integration.
- **Man pages:** Installs a man page (`man cec-controller`) when installed via `.deb` or `.rpm` package.

## Installation

> [!IMPORTANT]  
> Make sure to have no other services running libcec or any kind of cec-client because only one process can lock the CEC serial.

### From Release (Recommended)

Download pre-built binaries and packages from the [Releases](https://github.com/eliottness/cec-controller/releases) page:

**Ubuntu/Debian (amd64):**
```sh
wget https://github.com/eliottness/cec-controller/releases/latest/download/cec-controller_<version>_ubuntu_amd64.deb
sudo dpkg -i cec-controller_<version>_ubuntu_amd64.deb
```

**Ubuntu/Debian (arm64):**
```sh
wget https://github.com/eliottness/cec-controller/releases/latest/download/cec-controller_<version>_ubuntu_arm64.deb
sudo dpkg -i cec-controller_<version>_ubuntu_arm64.deb
```

**Fedora/RHEL (amd64):**
```sh
wget https://github.com/eliottness/cec-controller/releases/latest/download/cec-controller_<version>_fedora_amd64.rpm
sudo dnf install cec-controller_<version>_fedora_amd64.rpm
```

**Fedora/RHEL (arm64):**
```sh
wget https://github.com/eliottness/cec-controller/releases/latest/download/cec-controller_<version>_fedora_arm64.rpm
sudo dnf install cec-controller_<version>_fedora_arm64.rpm
```

### From Source

#### Prerequisites

- Go 1.24+
- [libcec](https://libcec.pulse-eight.com/) (`sudo apt install libcec-dev`)
- Linux with uinput support (for virtual keyboard)

#### Build

Requires `libcec-dev` and `libp8-platform-dev` on debian-based systems or just `libcec-devel` on fedora-based systems:

```sh
go build -o cec-controller .
```

## Usage

```sh
./cec-controller [flags]
```

### Configuration

cec-controller can be configured via command-line flags or a YAML configuration file. Command-line flags take precedence over the configuration file.

#### Configuration File

Create a configuration file at `/etc/cec-controller.yaml`. See [`cec-controller.yaml.example`](cec-controller.yaml.example) for a complete example.

```yaml
# Example configuration
cec-adapter: "/dev/ttyACM0"
device-name: "My PC"
debug: false
retries: 5
keymap:
  "1": "29+2"    # CEC key 1 -> Ctrl+1
  "2": "29+3"    # CEC key 2 -> Ctrl+2
devices:
  - "0"
  - "1"
```

### Common Flags

- `--cec-adapter=<path>`  
  Path to HDMI-CEC adapter e.g. /dev/ttyACM0. Leave blank for auto-detect.

- `--debug`  
  Enable debug logging.

- `--keymap <cec>:<linux>`  
  Add or override CEC to Linux key mappings (repeat as needed). Example: `--keymap 1:105` maps CEC key `1` to Linux key
  code `105` (KEY_KP1). You can also specify modifier keys using `+`, e.g. `--keymap 1:29+105` maps CEC key `1` to Ctrl+KP1.

- `--no-power-events`  
  Disable handling of system power events.

- `--devices`
  Power event device logical addresses (e.g. --devices 0,1). Defaults to 0.

- `--retries`
  Number of times to retry opening the CEC adapter on failure. Default is 5. Each attempt may take up to 10 seconds.

- `--restart-retries`
  Maximum number of process restarts when the CEC library gets stuck. Default is 3. Set to 0 to disable restarts.

- `--device-name`
  Device name to report to the CEC network. Default is the hostname.

- `--set-active-source`
  Claim the active HDMI source on startup, causing the TV to switch its input to this device.

- `--active-source-type`
  CEC device type to report when claiming active source. Default is `4` (Playback Device, suitable for PCs).
  Accepted values: `0`=TV, `1`=Recording, `3`=Tuner, `4`=Playback, `5`=AudioSystem.

#### Example using custom key mappings

Key mapping data for CEC can be found [here](https://github.com/claes/cec/blob/6db0712de894ea0c026b023b02181fee00babd39/cec.go#L147)

Linux key codes can be found [here](https://sites.uclouvain.be/SystInfo/usr/include/linux/input.h.html)

Here for example, CEC buttons "1" and "2" are mapped to Ctrl+1 and Ctrl+2 to use Steam Big Picture overlays from my TV remote:

```sh
./cec-controller --keymap 1:29+2 --keymap 2:29+3
```

## Systemd Integration

See [`cec-controller.service`](cec-controller.service):

```ini
[Unit]
Description=CEC Controller CLI Service
After=local-fs.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cec-controller
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Power Event Handling

This app detects and reacts to:

- **Startup:** Powers on connected devices when the service starts alongside systemd
- **Shutdown:** Puts connected devices to standby on system shutdown/reboot
- **Sleep/Resume:** Puts devices to standby on suspend, powers them on again on resume

Before putting devices to standby, cec-controller acquires a systemd-logind [delay inhibitor lock](https://systemd.io/INHIBITOR_LOCKS/) for `sleep` and `shutdown`. This guarantees the CEC standby command completes before the system proceeds, preventing TVs and receivers from being left powered on after the host sleeps.

## Contributing

PRs and issues are welcome!

### Running tests locally

```sh
# Install build dependencies (Ubuntu/Debian)
sudo apt-get install -y libcec-dev libp8-platform-dev

CGO_ENABLED=1 go test ./...
```

### Before submitting a PR

```sh
# Formatting (CI enforces this)
gofmt -l .        # should print nothing
gofmt -w .        # fix in place

# Vet
CGO_ENABLED=1 go vet ./...
```

CI will run `gofmt`, `go vet`, and `go test` automatically on every PR, then build binaries for amd64 and arm64.

## Releases

Releases are automated via GitHub Actions. Push a semver tag to trigger a build:

```sh
git tag v1.2.3
git push origin v1.2.3
```

The pipeline builds native binaries on `ubuntu-latest` (amd64) and `ubuntu-24.04-arm` (arm64), then packages them with goreleaser into `.tar.gz`, `.deb`, and `.rpm` artifacts published to the [Releases](https://github.com/eliottness/cec-controller/releases) page.

## License

GNU General Public License v3.0 or later. See [LICENSE](LICENSE) for details.
