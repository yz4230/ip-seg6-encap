# ip-seg6-encap

`ip-seg6-encap` is a small Go CLI for adding IPv4 or IPv6 routes that use SRv6 encapsulation and reserve extra zeroed TLV space in the Segment Routing Header.

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

There are currently no committed test files.

## Usage

Add an IPv4 route with SRv6 encapsulation and 8 reserved SRH bytes:

```sh
sudo ./build/ip-seg6-encap add \
  --prefix 10.4.0.0/24 \
  --segs fd00:a:7:0:8200::,fd00:a:3:0:8300::,fd00:a:6::1,fd00:a:4::d4 \
  --dev eth0 \
  --reserve 8
```

The same command can be run without building first:

```sh
sudo go run . add \
  --prefix 10.4.0.0/24 \
  --segs fd00:a:7:0:8200::,fd00:a:3:0:8300::,fd00:a:6::1,fd00:a:4::d4 \
  --dev eth0 \
  --reserve 8
```

### Flags

| Flag | Required | Description |
| --- | --- | --- |
| `--prefix` | yes | IPv4 or IPv6 destination prefix to install. |
| `--segs` | yes | One or more SRv6 segment IPv6 addresses. |
| `--dev` | yes | Network interface used for the route link index. |
| `--reserve` | no | Number of bytes to reserve as zeroed SRH TLV space. Defaults to `0`. Must be a multiple of `8`. |

Multiple segments can be passed as a comma-separated value accepted by Cobra's IP slice flag parser:

```sh
sudo ./build/ip-seg6-encap add \
  --prefix 2001:db8:100::/64 \
  --segs 2001:db8::1,2001:db8::2 \
  --dev eth0 \
  --reserve 16
```

## Behavior

The `add` command:

1. accepts an IPv4 or IPv6 `--prefix`,
2. resolves the interface named by `--dev`,
3. builds an SRv6 encapsulation route with the requested segment list, encoding
   the SRH segment array in Linux kernel order,
4. appends `--reserve` bytes of zeroed TLV space to the encoded SRH,
5. installs the route through netlink, replacing an existing route when the kernel reports `EEXIST`.

SRH encoding fails when no segments are provided, when `--reserve` is not a multiple of `8`, or when the resulting SRH length exceeds the kernel field size.

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
