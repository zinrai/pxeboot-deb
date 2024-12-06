# PXE Boot Server and Configuration Tools

A suite of tools for automating OS installation over the network using PXE boot. This repository provides a complete solution through three complementary components.

## Architecture Overview

1. `pxeboot-server`: Base server setup (DHCP, TFTP, HTTP)
  -  Ansible playbook that configures essential network services (DHCP, TFTP, HTTP) and network settings for PXE booting
2. `pxebootstrap`: Downloads ISOs and creates boot menus
  - Command-line tool that prepares the boot environment by downloading installation media and generating boot menus
3. `pxeboot-api`: Manages client-specific configurations
  - HTTP API server that handles dynamic, client-specific boot configurations based on MAC addresses

## Setup Workflow

1. Configure and deploy base PXE server:
   ```bash
   $ cd pxeboot-server
   $ ansible-playbook -i inventory pxeboot-server.yml
   ```

2. Initialize boot environment:
   ```bash
   $ cd pxebootstrap
   $ go build
   $ sudo ./pxebootstrap -config config.yaml
   ```

3. Deploy configuration API:
   ```bash
   $ cd pxeboot-api
   $ go build -o pxeboot-api cmd/server/main.go
   $ sudo ./pxeboot-api
   ```

For detailed configuration and usage of each component, please refer to their respective README files:

- [pxeboot-server/README.md](pxeboot-server/README.md)
- [pxebootstrap/README.md](pxebootstrap/README.md)
- [pxeboot-api/README.md](pxeboot-api/README.md)

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
