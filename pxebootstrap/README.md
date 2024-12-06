# PXE Boot Configuration Bootstrap Tool

A tool to configure PXE boot environments for network installation of operating systems. It downloads ISO files and generates both PXELinux and iPXE boot menus.

## Features

- Downloads installation ISO files
- Generates PXELinux and iPXE boot menus
- Supports multiple operating system targets
- Supports ISO file reuse with optional update capability

## Prerequisites

[pxeboot-server](../pxeboot-server)

## Installation

```bash
$ go build
```

## Configuration

Create a YAML configuration file (e.g., `config.yaml`):

```yaml
tftpboot_dir: "/var/www/tftpboot"
iso_dir: "/var/www/iso"
pxe_server_host: "192.168.10.1"
targets:
  - name: "debian"
    codename: "bookworm"
    iso_file: "https://deb.debian.org/debian/dists/bullseye/main/installer-amd64/current/images/netboot/mini.iso"
```

### Configuration Options

- `tftpboot_dir`: TFTP root directory path
- `iso_dir`: Directory to store downloaded ISO files
- `pxe_server_host`: PXE server IP address or hostname
- `targets`: List of operating system targets
  - `name`: Operating system name
  - `codename`: Release codename
  - `iso_file`: URL to the ISO file

## Usage

Basic usage:

```bash
$ sudo ./pxebootstrap -config config.yaml
```

To update existing ISO files:

```bash
$ sudo ./pxebootstrap -config config.yaml --update-iso
```

### Directory Structure

The tool creates the following directory structure:

```
/var/www/
├── iso/
│   └── images/
│       └── <name>/
│           └── <codename>/
│               └── <iso-file>
└── tftpboot/
    ├── bios/
    │   └── pxelinux.cfg/
    │       └── default
    └── ipxe/
        └── boot.ipxe
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
