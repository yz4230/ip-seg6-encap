# ip-seg6-encap

`ip-seg6-encap` is a small Go CLI for adding IPv4 or IPv6 routes that use SRv6 encapsulation and optional TLV space in the Segment Routing Header.

The tool uses Linux netlink to install a static route with `SEG6_IPTUN_MODE_ENCAP`. If the route already exists, it replaces the route with the requested SRv6 encapsulation settings.

## Requirements

- Linux with SRv6 support enabled in the kernel
- Go 1.26.3 or newer, as declared in `go.mod`
- Privileges to change routes, such as root or `CAP_NET_ADMIN`
- A target network interface or network namespace for the route

For manual testing, prefer a disposable network namespace or test interface instead of changing host routes directly.

## Build

Build all packages:

```sh
go build ./...
```

Build a CLI binary in `build/`:

```sh
make build
```

## Test

Run the package tests:

```sh
go test ./...
```

## Usage

Add an IPv4 route with SRv6 encapsulation and an initialized SRH TLV slot:

```sh
sudo ./build/ip-seg6-encap add \
  --prefix 10.5.0.0/24 \
  --segs fd00:a:3:0:8200::,fd00:a:4:0:8300::,fd00:a:7::1,fd00:a:5::d4 \
  --dev eth0 \
  --tlv-type 252 \
  --tlv-len 1
```

The same command can be run without building first:

```sh
sudo go run . add \
  --prefix 10.5.0.0/24 \
  --segs fd00:a:3:0:8200::,fd00:a:4:0:8300::,fd00:a:7::1,fd00:a:5::d4 \
  --dev eth0 \
  --tlv-type 252 \
  --tlv-len 1
```

### Flags

| Flag | Required | Description |
| --- | --- | --- |
| `--prefix` | yes | IPv4 or IPv6 destination prefix to install. |
| `--segs` | yes | One or more SRv6 segment IPv6 addresses. |
| `--dev` | yes | Network interface used for the route link index. |
| `--tlv-type` | no | SRH TLV type to initialize. Must be specified with `--tlv-len`. Range is `1-255`; `0` is Pad1 and is rejected for ordinary TLVs. |
| `--tlv-len` | no | SRH TLV value length to initialize. Must be specified with `--tlv-type`. Range is `0-255`; the value bytes are zero-filled. |

Multiple segments can be passed as a comma-separated value accepted by Cobra's IP slice flag parser:

```sh
sudo ./build/ip-seg6-encap add \
  --prefix 2001:db8:100::/64 \
  --segs 2001:db8::1,2001:db8::2 \
  --dev eth0 \
  --tlv-type 252 \
  --tlv-len 1
```

## Behavior

The `add` command:

1. accepts an IPv4 or IPv6 `--prefix`,
2. resolves the interface named by `--dev`,
3. builds an SRv6 encapsulation route with the requested segment list, encoding
   the SRH segment array in Linux kernel order,
4. appends initialized TLV space when `--tlv-type/--tlv-len` is provided,
5. installs the route through netlink, replacing an existing route when the kernel reports `EEXIST`.

In TLV mode, the encoded TLV is `type,len,zero-filled value` followed by Pad1 or PadN so the total TLV area is aligned to 8 bytes. For example, `--tlv-type 252 --tlv-len 1` encodes `fc 01 00 04 03 00 00 00`.

SRH encoding fails when no segments are provided, when TLV options are incomplete or invalid, or when the resulting SRH length exceeds the kernel field size.

## Development

Format edited Go files before committing:

```sh
gofmt -w main.go cmd/*.go
```

Useful validation commands:

```sh
go test ./...
go build ./...
```

Do not commit generated binaries, local environment files, or machine-specific network configuration.
