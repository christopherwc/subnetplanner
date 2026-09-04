# Subnet Planner

A desktop GUI tool, written in Go, for planning IPv4 and IPv6 subnets.

## Features

- **Subnet Info** — enter any CIDR (IPv4 or IPv6) and see its network address,
  first/last usable address, broadcast address (IPv4), and total/usable host
  counts.
- **Equal Split** — split a base network into a number of equal-sized
  subnets, either by target subnet count or by target prefix length.
- **VLSM Planner** — allocate variable-length subnets from a base network
  to a list of named host-count requirements (e.g. `Sales,50`), packed
  automatically for minimal waste.

Full IPv6 support is implemented with 128-bit arithmetic (`math/big`), so
plans covering huge address spaces (e.g. splitting a `/48` into `/64`s) work
correctly.

## Building and running

Requires Go 1.22+. On Linux, the GUI toolkit ([Fyne](https://fyne.io/)) needs
OpenGL/X11 development headers to build:

```sh
sudo apt-get install libgl1-mesa-dev xorg-dev
```

Then:

```sh
go run ./cmd/subnetplanner-gui
```

## Project layout

- `internal/subnet` — address-family-agnostic subnet planning engine
  (parsing, splitting, VLSM allocation). No GUI dependencies.
- `internal/guiapp` — Fyne widget wiring on top of `internal/subnet`.
- `cmd/subnetplanner-gui` — application entry point.

## Testing

```sh
go test ./... -cover
```

The `internal/subnet` and `internal/guiapp` packages (all of the tool's
logic) are covered at 100% by unit tests, including GUI interaction tests
run against Fyne's headless test driver (no display required). The two-line
`main()` in `cmd/subnetplanner-gui`, which just wires the tested app package
to a real, blocking event loop, is intentionally left untested.
