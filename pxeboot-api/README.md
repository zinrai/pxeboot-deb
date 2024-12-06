# PXE Boot Configuration HTTP API

This HTTP API server automates OS installation by generating MAC address-specific PXE boot configurations. It enables network booting with customized settings for each machine, supporting both BIOS and iPXE boot environments. The server is specifically designed to streamline the process of deploying operating systems across multiple machines while maintaining individual configurations.

## Features

- Generates configuration files for:
  - BIOS PXE boot ( pxelinux )
  - iPXE boot ( ipxe )
  - dnsmasq
- Lists available ISO files
- Supports MAC address-specific configurations
- HTTP-based API interface

## Prerequisites

[pxeboot-server](../pxeboot-server)

## Directory Structure

The ISO directory should follow this structure:

```
/var/www/iso/images/
└── debian
    └── bookworm
        └── mini.iso
```

## Installation

```bash
$ go build -o pxeboot-api cmd/server/main.go
```

## Configuration

Edit the constants in `config/config.go` to match your environment:

```go
const (
    TFTPBootDir   = "/var/www/tftpboot"
    ISODir        = "/var/www/iso/images"
    PXEServerHost = "192.168.10.1"
)
```

## API Endpoints

### Generate Configuration (`POST /generate-config`)

Generates PXE boot configuration files for a specific MAC address.

Request:

```json
{
    "mac_address": "52:54:00:3f:45:e5",
    "name": "debian",
    "codename": "bookworm",
    "iso": "mini.iso"
}
```

Response:

```json
{
    "status": "success",
    "message": "Configuration files generated successfully",
    "files": {
        "pxelinux_config": "/var/www/tftpboot/bios/pxelinux.cfg/01-52-54-00-3f-45-e5",
        "ipxe_config": "/var/www/tftpboot/ipxe/01-52-54-00-3f-45-e5",
        "dnsmasq_config": "/etc/dnsmasq.d/99-fixip-52-54-00-3f-45-e5.conf"
    }
}
```

### List Available ISOs (`GET /list-isos`)

Lists all available ISO files in the configured directory.

Response:

```json
{
    "status": "success",
    "isos": [
        {
            "name": "debian",
            "codename": "bookworm",
            "filename": "mini.iso"
        }
    ]
}
```

## Usage Example

Start the server:

```bash
sudo ./pxeboot-api
```

Generate configuration:

```
$ curl -X POST http://localhost:8080/generate-config \
  -H "Content-Type: application/json" \
  -d '{
      "mac_address": "52:54:00:3f:45:e5",
      "name": "debian",
      "codename": "bookworm",
      "iso": "mini.iso"
  }'
```

List available ISOs:

```
$ curl http://localhost:8080/list-isos
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
