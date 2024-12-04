package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

type BootFiles struct {
	Vmlinuz string `yaml:"vmlinuz"`
	Initrd  string `yaml:"initrd"`
}

type Target struct {
	Name      string    `yaml:"name"`
	Codename  string    `yaml:"codename"`
	Version   string    `yaml:"version"`
	ISOFile   string    `yaml:"iso_file"`
	BootFiles BootFiles `yaml:"boot_files"`
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
	"base": filepath.Base,
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	log.Printf("Reading configuration from: %s\n", configPath)

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Found %d targets to process", len(config.Targets))

	if err := setupBootFiles(config); err != nil {
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

func setupBootFiles(config *Config) error {
	for i, target := range config.Targets {
		log.Printf("Processing target %d/%d: %s %s %s",
			i+1, len(config.Targets), target.Name, target.Codename, target.Version)

		if err := processTarget(config, target); err != nil {
			return fmt.Errorf("failed to process target %s: %v", target.Name, err)
		}
	}

	// Generate PXE menus after processing all targets
	if err := generatePXEMenus(config); err != nil {
		return fmt.Errorf("failed to generate PXE menus: %v", err)
	}

	return nil
}

func processTarget(config *Config, target Target) error {
	// Create directory paths
	isoPath := filepath.Join(config.ISODir, "images", target.Name, target.Codename, target.Version)
	tftpPath := filepath.Join(config.TFTPBootDir, "images", target.Name, target.Codename, target.Version)
	mountPoint := filepath.Join("/mnt", target.Name, target.Codename, target.Version)

	// Create ISO directory
	if err := os.MkdirAll(isoPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", isoPath, err)
	}
	log.Printf("Created directory: %s", isoPath)

	// Download ISO
	isoFilePath := filepath.Join(isoPath, filepath.Base(target.ISOFile))
	if err := downloadISO(target.ISOFile, isoFilePath); err != nil {
		return err
	}

	// Mount and copy files only if boot_files is set
	if target.BootFiles.Vmlinuz != "" && target.BootFiles.Initrd != "" {
		// Create tftp and mount directories
		for _, dir := range []string{tftpPath, mountPoint} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", dir, err)
			}
			log.Printf("Created directory: %s", dir)
		}

		// Mount ISO and copy files
		if err := mountAndCopyFiles(isoFilePath, tftpPath, mountPoint, target.BootFiles); err != nil {
			return err
		}
	} else {
		log.Printf("Skipping mount and copy for %s (boot_files not configured)", target.Name)
	}

	return nil
}

func downloadISO(url, destPath string) error {
	if _, err := os.Stat(destPath); err == nil {
		log.Printf("ISO file already exists at %s", destPath)
		return nil
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

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func mountAndCopyFiles(isoPath, tftpPath, mountPoint string, bootFiles BootFiles) error {
	// Check if directory is empty (not mounted)
	empty, err := isDirEmpty(mountPoint)
	if err != nil {
		return fmt.Errorf("failed to check mount point: %v", err)
	}

	if empty {
		// Mount ISO
		log.Printf("Mounting ISO %s to %s", isoPath, mountPoint)
		cmd := exec.Command("mount", "-o", "loop", isoPath, mountPoint)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to mount ISO: %v, output: %s", err, string(output))
		}
		defer func() {
			log.Printf("Unmounting %s", mountPoint)
			cmd := exec.Command("umount", mountPoint)
			if err := cmd.Run(); err != nil {
				log.Printf("Warning: failed to unmount ISO: %v", err)
			}
		}()
	} else {
		log.Printf("Mount point %s is already mounted or contains files", mountPoint)
	}

	// Copy boot files
	files := []struct{ src, dest string }{
		{
			src:  filepath.Join(mountPoint, bootFiles.Vmlinuz),
			dest: filepath.Join(tftpPath, "vmlinuz"),
		},
		{
			src:  filepath.Join(mountPoint, bootFiles.Initrd),
			dest: filepath.Join(tftpPath, "initrd"),
		},
	}

	for _, file := range files {
		log.Printf("Copying %s to %s", file.src, file.dest)
		if err := copyFile(file.src, file.dest); err != nil {
			return fmt.Errorf("failed to copy %s: %v", file.src, err)
		}
	}

	return nil
}

func generatePXEMenus(config *Config) error {
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
	log.Printf("Loading template from: %s", tplPath)

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

	log.Printf("Generated PXE menu at: %s", defaultFile)
	return nil
}

func copyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, input, 0644)
}
