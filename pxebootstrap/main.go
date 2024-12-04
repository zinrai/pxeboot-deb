package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Target struct {
	Name     string `yaml:"name"`
	Codename string `yaml:"codename"`
	ISOFile  string `yaml:"iso_file"`
}

type Config struct {
	TFTPBootDir   string   `yaml:"tftpboot_dir"`
	ISODir        string   `yaml:"iso_dir"`
	PXEServerHost string   `yaml:"pxe_server_host"`
	Targets       []Target `yaml:"targets"`
}

type MenuData struct {
	PXEServerHost string
	Targets       []Target
}

var templateFuncs = template.FuncMap{
	"base":  filepath.Base,
	"title": strings.Title,
}

func main() {
	var configPath string
	var updateISO bool
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.BoolVar(&updateISO, "update-iso", false, "Update ISO files even if they exist")
	flag.Parse()

	log.Printf("Reading configuration from: %s\n", configPath)

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Found %d targets to process", len(config.Targets))

	if err := setupBootFiles(config, updateISO); err != nil {
		log.Fatal(err)
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate configuration
	if len(config.Targets) == 0 {
		return nil, fmt.Errorf("no targets found in configuration")
	}

	return &config, nil
}

func setupBootFiles(config *Config, updateISO bool) error {
	for i, target := range config.Targets {
		log.Printf("Processing target %d/%d: %s %s",
			i+1, len(config.Targets), target.Name, target.Codename)

		if err := processTarget(config, target, updateISO); err != nil {
			return fmt.Errorf("failed to process target %s: %v", target.Name, err)
		}
	}

	// Generate boot menus after processing all targets
	if err := generateBootMenus(config); err != nil {
		return fmt.Errorf("failed to generate boot menus: %v", err)
	}

	return nil
}

func processTarget(config *Config, target Target, updateISO bool) error {
	// Create directory paths
	isoPath := filepath.Join(config.ISODir, "images", target.Name, target.Codename)

	// Create ISO directory
	if err := os.MkdirAll(isoPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", isoPath, err)
	}
	log.Printf("Created directory: %s", isoPath)

	// Download ISO
	isoFilePath := filepath.Join(isoPath, filepath.Base(target.ISOFile))
	if err := downloadISO(target.ISOFile, isoFilePath, updateISO); err != nil {
		return err
	}

	log.Printf("Processing completed for %s %s", target.Name, target.Codename)
	return nil
}

func downloadISO(url, destPath string, updateISO bool) error {
	if !updateISO {
		if _, err := os.Stat(destPath); err == nil {
			log.Printf("ISO file already exists at %s (use --update-iso to update)", destPath)
			return nil
		}
	} else {
		log.Printf("ISO update enabled, downloading ISO...")
	}

	log.Printf("Downloading ISO from %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download ISO: %v", err)
	}
	defer resp.Body.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create ISO file: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write ISO file: %v", err)
	}

	log.Printf("Downloaded ISO to %s", destPath)
	return nil
}

func generateBootMenus(config *Config) error {
	// PXELinux menu generation
	if err := generatePXELinuxMenu(config); err != nil {
		return fmt.Errorf("failed to generate PXELinux menu: %v", err)
	}

	// iPXE menu generation
	if err := generateIPXEMenu(config); err != nil {
		return fmt.Errorf("failed to generate iPXE menu: %v", err)
	}

	return nil
}

func generatePXELinuxMenu(config *Config) error {
	biosPath := filepath.Join(config.TFTPBootDir, "bios")
	pxelinuxCfgPath := filepath.Join(biosPath, "pxelinux.cfg")

	// Create symlink for images directory
	imagesPath := filepath.Join(config.TFTPBootDir, "images")
	osImagesPath := filepath.Join(biosPath, "images")

	// Force create symlink
	_ = os.Remove(osImagesPath)
	if err := os.Symlink(imagesPath, osImagesPath); err != nil {
		return fmt.Errorf("failed to create symlink: %v", err)
	}
	log.Printf("Created symlink from %s to %s", imagesPath, osImagesPath)

	// Get current execution directory
	execDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	// Build absolute paths of template files
	tplPath := filepath.Join(execDir, "templates", "pxe_menu.tpl")
	log.Printf("Loading PXELinux template from: %s", tplPath)

	tplContent, err := os.ReadFile(tplPath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	tpl, err := template.New("pxe_menu").Funcs(templateFuncs).Parse(string(tplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	data := MenuData{
		PXEServerHost: config.PXEServerHost,
		Targets:       config.Targets,
	}

	defaultFile := filepath.Join(pxelinuxCfgPath, "default")
	f, err := os.Create(defaultFile)
	if err != nil {
		return fmt.Errorf("failed to create default file: %v", err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	log.Printf("Generated PXELinux menu at: %s", defaultFile)
	return nil
}

func generateIPXEMenu(config *Config) error {
	execDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	ipxePath := filepath.Join(config.TFTPBootDir, "ipxe")
	if err := os.MkdirAll(ipxePath, 0755); err != nil {
		return fmt.Errorf("failed to create ipxe directory: %v", err)
	}

	tplPath := filepath.Join(execDir, "templates", "ipxe_menu.tpl")
	log.Printf("Loading iPXE template from: %s", tplPath)

	tplContent, err := os.ReadFile(tplPath)
	if err != nil {
		return fmt.Errorf("failed to read iPXE template file: %v", err)
	}

	tpl, err := template.New("ipxe_menu").Funcs(templateFuncs).Parse(string(tplContent))
	if err != nil {
		return fmt.Errorf("failed to parse iPXE template: %v", err)
	}

	data := MenuData{
		PXEServerHost: config.PXEServerHost,
		Targets:       config.Targets,
	}

	bootIpxe := filepath.Join(ipxePath, "boot.ipxe")
	f, err := os.Create(bootIpxe)
	if err != nil {
		return fmt.Errorf("failed to create boot.ipxe: %v", err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute iPXE template: %v", err)
	}

	log.Printf("Generated iPXE menu at: %s", bootIpxe)
	return nil
}
