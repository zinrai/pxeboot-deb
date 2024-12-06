# PXE Boot Server Setup

Ansible Playbook to set up a PXE boot server on Debian that supports both BIOS and EFI booting.

## Features

- Supports both BIOS (Legacy) and EFI boot
- Uses iPXE for EFI boot with enhanced features
- HTTP boot support for faster file transfer
- Configurable DHCP settings
- IP masquerading for client internet access

## Requirements

- Debian GNU/Linux
- Two network interfaces:
  - internet access ( WAN )
  - PXE boot clients ( LAN )
- Ansible

## Quick Start

Run the playbook:

```bash
$ ansible-playbook -i inventory pxeboot-server.yml
```

## Configuration

### Network Configuration

Edit `group_vars/all.yml` to set up your network configuration:

```yaml
dnsmasq:
  interface: "ens8"          # LAN interface
  router_ip: "192.168.10.1"  # PXE server IP
  netmask: "255.255.255.0"
  dns_server: "192.168.10.1"
  dhcp_range: "192.168.10.100,192.168.10.200,12h"

nftables:
  wan_interface: "ens3"      # WAN interface

pxeboot_server_ip: "192.168.10.1"
```

## Directory Structure

```
/var/www/tftpboot/
├── bios/                  # BIOS boot files
├── ipxe/                  # iPXE boot files
├── images/                # Kernel and initrd files
└── grub/                  # GRUB configuration
```

## Ansible Roles

- `dnsmasq_conf`: DHCP and TFTP server configuration
- `nginx_conf`: HTTP server setup
- `boot_menu_conf`: PXE Boot menu configuration
- `nftables_conf`: IP masquerading setup

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
