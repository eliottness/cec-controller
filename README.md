# cec-controller

Use libcec and dbus to start and stop your TV and simulate keyboard presses from your TV Remote

## Building

Build the application using Go:

```bash
go build -o cec-controller .
```

## Usage

```bash
# Show help
./cec-controller -help

# Show version
./cec-controller -version

# Power on the TV
./cec-controller -power-on

# Power off the TV
./cec-controller -power-off

# Get TV status
./cec-controller -status
```

## Installation

### Manual Installation

```bash
# Build the binary
go build -o cec-controller .

# Copy to system path
sudo cp cec-controller /usr/local/bin/

# Install systemd service
sudo cp cec-controller.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable cec-controller
sudo systemctl start cec-controller
```

### Using Pre-built Binaries

Download the latest release from the [releases page](https://github.com/eliottness/cec-controller/releases) for your platform (Linux, macOS, or Windows).

## Development

The project uses GitHub Actions to automatically build binaries for multiple platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

Binaries are built on every push and pull request, and automatically attached to releases.
