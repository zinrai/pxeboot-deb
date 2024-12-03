package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	TFTPBootDir = "/var/www/tftpboot"
)

type HostConfig struct {
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname"`
	Linux      string `json:"linux"`
	Codename   string `json:"codename"`
	Version    string `json:"version"`
}

func (c *HostConfig) CheckRequiredFiles() error {
	files := []string{
		filepath.Join(TFTPBootDir, "images", c.Linux, c.Codename, c.Version, "vmlinuz"),
		filepath.Join(TFTPBootDir, "images", c.Linux, c.Codename, c.Version, "initrd"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required file not found: %s", file)
		}
	}

	return nil
}

func (c *HostConfig) GetMACForFilename() string {
	return filepath.Join(
		fmt.Sprintf("fixip-%s-%s",
			c.Hostname,
			strings.ReplaceAll(c.MACAddress, ":", "-")),
	)
}
