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
	TFTPBootDir string   `yaml:"tftpboot_dir"`
	ISODir      string   `yaml:"iso_dir"`
	Targets     []Target `yaml:"targets"`
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
	return nil
}

func processTarget(config *Config, target Target) error {
	// Create directory paths
	isoPath := filepath.Join(config.ISODir, "images", target.Name, target.Codename, target.Version)
	tftpPath := filepath.Join(config.TFTPBootDir, "images", target.Name, target.Codename, target.Version)
	mountPoint := filepath.Join("/mnt", target.Name, target.Codename, target.Version)

	// Create directories
	for _, dir := range []string{isoPath, tftpPath, mountPoint} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
		log.Printf("Created directory: %s", dir)
	}

	// Download ISO
	isoFilePath := filepath.Join(isoPath, filepath.Base(target.ISOFile))
	if err := downloadISO(target.ISOFile, isoFilePath); err != nil {
		return err
	}

	// Mount ISO and copy files
	if err := mountAndCopyFiles(isoFilePath, tftpPath, mountPoint, target.BootFiles); err != nil {
		return err
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

func copyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, input, 0644)
}