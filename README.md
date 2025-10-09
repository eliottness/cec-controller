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

### Prerequisites

- Go 1.21+
- [libcec](https://libcec.pulse-eight.com/) (`sudo apt install libcec-dev`)
- Linux with uinput support (for virtual keyboard)

### Build

```sh
go build -o cec-controller main.go
```

Or use the provided [GitHub Actions workflow](.github/workflows/release.yml) for automated builds.

## Usage

```sh
./cec-controller [flags]
```

### Common Flags

- `--cec-adapter=<path>`  
  Path to HDMI-CEC adapter e.g. /dev/ttyACM0. Leave blank for auto-detect.

- `--debug`  
  Enable debug logging.

- `--keymap <cec>:<linux>`  
  Add or override CEC to Linux key mappings (repeat as needed). Example: `--keymap 1:105` maps CEC key `1` to Linux key
  code `105` (KEY_KP1).

- `--no-power-events`  
  Disable handling of system power events.

- `--devices`  
  List available CEC adapters and exit. Comma-separated list of device ids. Defaults to zero.

#### Example

```sh
./cec-controller --debug --keymap 1:105 --keymap 2:106
```

## Systemd Integration

See [`cec-controller.service`](cec-controller.service):

```ini
[Unit]
Description=CEC Controller CLI Service
After=local-fs.target
DefaultDependencies=no

[Service]
Type=simple
ExecStart=/usr/local/bin/cec-controller
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Power Event Handling

This app detects and reacts to:

- **Startup:** Emitted when the service starts
- **Shutdown:** On system shutdown/reboot
- **Sleep/Resume:** On suspend/resume events

You can customize hooks for these events in your code.

## Contributing

PRs and issues are welcome! Please ensure code is formatted (`go fmt`) and tested.

## License

GNU General Public License v3.0 or later. See [LICENSE](LICENSE) for details.