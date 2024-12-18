package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zinrai/pxeboot-deb/pxeboot-api/config"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

var templateFiles = struct {
	PXELinux string
	IPXE     string
	Dnsmasq  string
}{
	PXELinux: "templates/pxelinux.cfg.tmpl",
	IPXE:     "templates/ipxeboot.tmpl",
	Dnsmasq:  "templates/dnsmasq.conf.tmpl",
}

func initLogger() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

func generateConfig(w http.ResponseWriter, r *http.Request) {
	InfoLogger.Printf("Received generate-config request from %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		ErrorLogger.Printf("Invalid method %s from %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg config.HostConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		ErrorLogger.Printf("Failed to decode JSON from %s: %v", r.RemoteAddr, err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	InfoLogger.Printf("Processing configuration for MAC: %s", cfg.MACAddress)

	// Verify the existence of the necessary files.
	if err := cfg.CheckRequiredFiles(); err != nil {
		ErrorLogger.Printf("Required file check failed for MAC %s: %v", cfg.MACAddress, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	macForFilename := cfg.GetMACForFilename()

	// Generate configuration files for both BIOS and iPXE
	files := map[string]string{
		"pxelinux_config": generateBIOSConfig(cfg, macForFilename),
		"ipxe_config":     generateIPXEConfig(cfg, macForFilename),
		"dnsmasq_config":  generateDnsmasqConfig(cfg, macForFilename),
	}

	for configType, filePath := range files {
		InfoLogger.Printf("Generating %s at %s", configType, filePath)
		if err := generateFromTemplate(configType, filePath, cfg); err != nil {
			ErrorLogger.Printf("Failed to generate %s for MAC %s: %v", configType, cfg.MACAddress, err)
			http.Error(w, fmt.Sprintf("Error generating %s: %v", configType, err), http.StatusInternalServerError)
			return
		}
		InfoLogger.Printf("Successfully generated %s", configType)
	}

	InfoLogger.Printf("Configuration generated successfully for MAC: %s", cfg.MACAddress)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Configuration files generated successfully",
		"files":   files,
	})
}

func generateFromTemplate(templateType, outputPath string, cfg config.HostConfig) error {
	var templatePath string
	switch templateType {
	case "pxelinux_config":
		templatePath = templateFiles.PXELinux
	case "ipxe_config":
		templatePath = templateFiles.IPXE
	case "dnsmasq_config":
		templatePath = templateFiles.Dnsmasq
	default:
		return fmt.Errorf("unknown template type: %s", templateType)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	return tmpl.Execute(f, cfg.GetTemplateData())
}

func generateBIOSConfig(cfg config.HostConfig, macForFilename string) string {
	return filepath.Join(
		config.TFTPBootDir,
		"bios/pxelinux.cfg",
		cfg.GetPXELinuxMACFormat(),
	)
}

func generateIPXEConfig(cfg config.HostConfig, macForFilename string) string {
	return filepath.Join(
		config.TFTPBootDir,
		"ipxe",
		cfg.GetPXELinuxMACFormat(),
	)
}

func generateDnsmasqConfig(cfg config.HostConfig, macForFilename string) string {
	return filepath.Join(
		"/etc/dnsmasq.d",
		fmt.Sprintf("%s.conf", macForFilename),
	)
}

func listAvailableISOs(w http.ResponseWriter, r *http.Request) {
	InfoLogger.Printf("Received list-isos request from %s", r.RemoteAddr)

	if r.Method != http.MethodGet {
		ErrorLogger.Printf("Invalid method %s from %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var isoList []config.ISOInfo

	err := filepath.Walk(config.ISODir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ErrorLogger.Printf("Error walking path %s: %v", path, err)
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(config.ISODir, path)
			if err != nil {
				ErrorLogger.Printf("Error getting relative path for %s: %v", path, err)
				return err
			}

			dir := filepath.Dir(relPath)
			name := filepath.Base(filepath.Dir(dir)) // ex: debian
			codename := filepath.Base(dir)           // ex: bookworm

			if name != "" && codename != "" {
				isoList = append(isoList, config.ISOInfo{
					Name:     name,
					Codename: codename,
					Filename: filepath.Base(path),
				})
				InfoLogger.Printf("Found ISO: %s/%s/%s", name, codename, filepath.Base(path))
			}
		}
		return nil
	})

	if err != nil {
		ErrorLogger.Printf("Failed to read ISO directory: %v", err)
		http.Error(w, fmt.Sprintf("Error reading ISO directory: %v", err), http.StatusInternalServerError)
		return
	}

	InfoLogger.Printf("Found %d ISO files", len(isoList))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"isos":   isoList,
	})
}

func main() {
	initLogger()
	InfoLogger.Printf("Starting PXEBoot API server")
	InfoLogger.Printf("TFTP Boot Directory: %s", config.TFTPBootDir)
	InfoLogger.Printf("ISO Directory: %s", config.ISODir)
	InfoLogger.Printf("PXE Server Host: %s", config.PXEServerHost)

	http.HandleFunc("/generate-config", generateConfig)
	http.HandleFunc("/list-isos", listAvailableISOs)

	InfoLogger.Printf("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		ErrorLogger.Fatalf("Server error: %v", err)
	}
}
