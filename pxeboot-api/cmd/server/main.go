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

var templateFiles = struct {
	PXELinux string
	Dnsmasq  string
}{
	PXELinux: "templates/pxelinux.tmpl",
	Dnsmasq:  "templates/dnsmasq.tmpl",
}

func generateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg config.HostConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Verify the existence of the necessary files.
	if err := cfg.CheckRequiredFiles(); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	macForFilename := cfg.GetMACForFilename()

	// Generate a configuration file.
	files := map[string]string{
		"pxelinux_config": generatePXELinuxConfig(cfg, macForFilename),
		"dnsmasq_config":  generateDnsmasqConfig(cfg, macForFilename),
	}

	for configType, filePath := range files {
		if err := generateFromTemplate(configType, filePath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("Error generating %s: %v", configType, err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Configuration files generated successfully",
		"files":   files,
	})
}

func listAvailableISOs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var isoList []config.ISOInfo

	err := filepath.Walk(config.ISODir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(config.ISODir, path)
			if err != nil {
				return err
			}

			parts := filepath.SplitList(filepath.Dir(relPath))
			if len(parts) >= 2 {
				isoList = append(isoList, config.ISOInfo{
					Linux:    parts[0],
					Codename: parts[1],
					Filename: filepath.Base(path),
				})
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading ISO directory: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"isos":   isoList,
	})
}

func generateFromTemplate(templateType, outputPath string, cfg config.HostConfig) error {
	var templatePath string
	switch templateType {
	case "pxelinux_config":
		templatePath = templateFiles.PXELinux
	case "dnsmasq_config":
		templatePath = templateFiles.Dnsmasq
	default:
		return fmt.Errorf("unknown template type: %s", templateType)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

func generatePXELinuxConfig(cfg config.HostConfig, macForFilename string) string {
	return filepath.Join(
		config.TFTPBootDir,
		"bios/pxelinux.cfg",
		cfg.GetPXELinuxMACFormat(),
	)
}

func generateDnsmasqConfig(cfg config.HostConfig, macForFilename string) string {
	return filepath.Join(
		"/etc/dnsmasq.d",
		fmt.Sprintf("%s.conf", macForFilename),
	)
}

func main() {
	http.HandleFunc("/generate-config", generateConfig)
	http.HandleFunc("/list-isos", listAvailableISOs)

	log.Printf("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
