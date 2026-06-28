cec - vendored libcec binding
=============================

This package is a **vendored, lightly modified** copy of the Go libcec binding
[`github.com/claes/cec`](https://github.com/claes/cec) (itself derived from
[`github.com/chbmuc/cec`](https://github.com/chbmuc/cec)), pinned at upstream
commit `6db0712de894`.

It is imported directly as part of this module
(`github.com/eliottness/cec-controller/cec`) rather than as an external
dependency, so that `cec-controller` can carry fixes upstream has not merged.

## Local modifications

* **Connection-loss notifications.** Upstream's `alert` callback only logged the
  event (`// TODO reconnect`). It now forwards every libcec alert to a new
  `Connection.Alerts` channel as a typed `*Alert`, so consumers can react to
  `AlertConnectionLost` — fired when the USB adapter drops off the bus and
  re-enumerates — instead of silently going deaf. See `callbacks.go` and the
  `AlertType` constants in `cec.go`.

Keep this list in sync when diverging further from upstream so the delta stays
easy to review or rebase.

## Build requirements

This package uses cgo against libcec, so the libcec headers must be installed at
build time (the C API is sourced via `pkg-config: libcec`):

* Fedora: `sudo dnf install libcec-devel`
* Debian/Ubuntu: `sudo apt-get install libcec-dev libp8-platform-dev`

The resulting binary links the matching `libcec.so` major version, so build on a
host whose libcec major version matches the deployment target.

## License

Upstream is MIT-licensed; the original license is retained verbatim in
[`LICENSE`](./LICENSE).
