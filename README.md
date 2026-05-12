# ip-seg6-encap

`ip-seg6-encap` is a small Go CLI for adding IPv6 routes that use SRv6 encapsulation and reserve extra zeroed TLV space in the Segment Routing Header.

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

Build a CLI binary in the repository root:

```sh
go build -o ip-sr-tlv .
```

The repository also includes a small Make target:

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

Add an IPv6 route with SRv6 encapsulation:

```sh
sudo ./ip-sr-tlv add \
  --prefix 2001:db8:100::/64 \
  --segs 2001:db8::1 \
  --dev eth0 \
  --reserve 8
```

The same command can be run without building first:

```sh
sudo go run . add \
  --prefix 2001:db8:100::/64 \
  --segs 2001:db8::1 \
  --dev eth0 \
  --reserve 8
```

### Flags

| Flag | Required | Description |
| --- | --- | --- |
| `--prefix` | yes | IPv6 destination prefix to install. IPv4 prefixes are rejected. |
| `--segs` | yes | One or more SRv6 segment IPv6 addresses. |
| `--dev` | yes | Network interface used for the route link index. |
| `--reserve` | no | Number of bytes to reserve as zeroed SRH TLV space. Defaults to `0`. Must be a multiple of `8`. |

Multiple segments can be passed as a comma-separated value accepted by Cobra's IP slice flag parser:

```sh
sudo ./ip-sr-tlv add \
  --prefix 2001:db8:100::/64 \
  --segs 2001:db8::1,2001:db8::2 \
  --dev eth0 \
  --reserve 16
```

## Behavior

The `add` command:

1. validates that `--prefix` is IPv6,
2. resolves the interface named by `--dev`,
3. builds an SRv6 encapsulation route with the requested segment list,
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
