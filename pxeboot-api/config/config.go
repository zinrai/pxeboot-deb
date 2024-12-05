package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	TFTPBootDir   = "/var/www/tftpboot"
	ISODir        = "/var/www/iso/images"
	PXEServerHost = "192.168.10.1"
)

type HostConfig struct {
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname"`
	Name       string `json:"name"`
	Codename   string `json:"codename"`
	ISOFile    string `json:"iso"`
}

type ISOInfo struct {
	Name     string `json:"name"`
	Codename string `json:"codename"`
	Filename string `json:"filename"`
}

func (c *HostConfig) CheckRequiredFiles() error {
	isoPath := filepath.Join(ISODir, c.Name, c.Codename, c.ISOFile)
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return fmt.Errorf("required file not found: %s", isoPath)
	}
	return nil
}

// Return configuration file for dnsmasq in MAC address format
func (c *HostConfig) GetMACForFilename() string {
	return filepath.Join(
		fmt.Sprintf("fixip-%s-%s",
			c.Hostname,
			strings.ReplaceAll(c.MACAddress, ":", "-")),
	)
}

// Returns MAC address format for PXELinux boot configuration file name
func (c *HostConfig) GetPXELinuxMACFormat() string {
	// Replace colons in MAC addresses with hyphens
	mac := strings.ToLower(strings.ReplaceAll(c.MACAddress, ":", "-"))
	return fmt.Sprintf("01-%s", mac)
}

// Auxiliary data for templates
func (c *HostConfig) GetTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"Name":          c.Name,
		"Codename":      c.Codename,
		"ISOFile":       c.ISOFile,
		"MACAddress":    c.MACAddress,
		"IPAddress":     c.IPAddress,
		"Hostname":      c.Hostname,
		"PXEServerHost": PXEServerHost,
	}
}
