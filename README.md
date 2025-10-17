# cec-controller

CEC Controller is a Linux CLI application that listens for HDMI-CEC key events and translates them to Linux virtual
keyboard actions. It also reacts to system power events (startup, shutdown, sleep, resume) and can be configured for
custom key mappings.

## Features

- **HDMI-CEC key listening:** Uses [libcec](https://libcec.pulse-eight.com/) via Go bindings to receive remote key
  presses from connected HDMI devices.
- **Virtual keyboard emulation:** Maps CEC keys to Linux key codes and triggers key events
  using [micmonay/keybd_event](https://github.com/micmonay/keybd_event). (See default key map [here](keymap.go))
- **Customizable key mapping:** Override or extend the CEC→Linux key map via CLI flags.
- **Power event hooks:** Responds to system startup, shutdown, sleep, and resume of the host machine and transmits
  corresponding CEC commands (e.g. "Power On", "Standby") to connected devices.
- **Systemd-ready:** Includes a sample systemd service file for robust startup and integration.

## Installation

> [!IMPORTANT]  
> Make sure to have no other services running libcec or any kind of cec-client because only one process can lock the CEC serial.

### From Release (Recommended)

Download pre-built binaries and packages from the [Releases](https://github.com/eliottness/cec-controller/releases) page:

**Ubuntu/Debian:**
```sh
# Download and install the .deb package
wget https://github.com/eliottness/cec-controller/releases/latest/download/cec-controller_<version>_ubuntu_amd64.deb
sudo dpkg -i cec-controller_<version>_ubuntu_amd64.deb
```

**Fedora/RHEL:**
```sh
# Download and install the .rpm package
wget https://github.com/eliottness/cec-controller/releases/latest/download/cec-controller-<version>-1.<dist>.x86_64.rpm
sudo dnf install cec-controller-<version>-1.<dist>.x86_64.rpm
```

### From Source

#### Prerequisites

- Go 1.24+
- [libcec](https://libcec.pulse-eight.com/) (`sudo apt install libcec-dev`)
- Linux with uinput support (for virtual keyboard)

#### Build

Requires `libcec-dev` and `libp8-platform-dev` on debian-based systems or just `libcec-devel` on fedora-based systems:

```sh
go build -o cec-controller main.go
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
  - "1:29+2"    # CEC key 1 -> Ctrl+1
  - "2:29+3"    # CEC key 2 -> Ctrl+2
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
  Power event device logical addresses (e.g. --devices 0,1). Default to 0

- `--retries`  
  Number of connection retries to the CEC adapter. Default is 5. Each try can take up to 10 seconds.

- `--device-name`
  Device name to report to CEC network. Default is the hostname

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

- **Startup:** Emitted when the service starts alongside systemd
- **Shutdown:** On system shutdown/reboot
- **Sleep/Resume:** On suspend/resume events

You can customize hooks for these events in your code.

## Contributing

PRs and issues are welcome! Please ensure code is formatted (`go fmt`) and tested.

## Releases

This project uses automated CI/CD workflows to build and release binaries for multiple platforms. See [RELEASE.md](RELEASE.md) for details on the release process.

## License

GNU General Public License v3.0 or later. See [LICENSE](LICENSE) for details.
