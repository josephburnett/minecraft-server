// Package main creates a .mcpack file from the behavior pack directory.
// An .mcpack is a ZIP file that can be double-clicked to import into Minecraft.
package main

import (
	"archive/zip"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Manifest represents the manifest.json structure
type Manifest struct {
	FormatVersion int            `json:"format_version"`
	Header        ManifestHeader `json:"header"`
	Modules       []Module       `json:"modules"`
	Dependencies  []Dependency   `json:"dependencies"`
}

type ManifestHeader struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	UUID             string `json:"uuid"`
	Version          [3]int `json:"version"`
	MinEngineVersion [3]int `json:"min_engine_version"`
}

type Module struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid"`
	Version  [3]int `json:"version"`
	Language string `json:"language,omitempty"`
	Entry    string `json:"entry,omitempty"`
}

type Dependency struct {
	ModuleName string `json:"module_name"`
	Version    string `json:"version"`
}

func main() {
	packDir := flag.String("pack", "bedrock/behavior_packs/burnodd_scripts", "Path to behavior pack directory")
	outputDir := flag.String("output-dir", "output", "Output directory for .mcpack file")
	noBump := flag.Bool("no-bump", false, "Skip version bump")
	flag.Parse()

	// Bump version and get the new version
	var version [3]int
	var err error
	if !*noBump {
		version, err = bumpVersion(*packDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error bumping version: %v\n", err)
			os.Exit(1)
		}
	} else {
		version, err = getVersion(*packDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading version: %v\n", err)
			os.Exit(1)
		}
	}

	versionStr := fmt.Sprintf("%d.%d.%d", version[0], version[1], version[2])
	fmt.Printf("Version: %s\n", versionStr)

	// Delete old .mcpack files
	if err := deleteOldPacks(*outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning old packs: %v\n", err)
		os.Exit(1)
	}

	// Create new pack with version in filename
	outputPath := filepath.Join(*outputDir, fmt.Sprintf("Burnodd-%s.mcpack", versionStr))
	if err := createMcpack(*packDir, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s\n", outputPath)
}

// generateUUID generates a random UUID v4
func generateUUID() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)

	// Set version (4) and variant (2) bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

func getVersion(packDir string) ([3]int, error) {
	manifestPath := filepath.Join(packDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return [3]int{}, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return [3]int{}, fmt.Errorf("parsing manifest: %w", err)
	}

	return manifest.Header.Version, nil
}

func bumpVersion(packDir string) ([3]int, error) {
	manifestPath := filepath.Join(packDir, "manifest.json")

	// Read manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return [3]int{}, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return [3]int{}, fmt.Errorf("parsing manifest: %w", err)
	}

	// Bump patch version
	manifest.Header.Version[2]++

	// Update description to include version
	versionStr := fmt.Sprintf("%d.%d.%d", manifest.Header.Version[0], manifest.Header.Version[1], manifest.Header.Version[2])
	manifest.Header.Description = fmt.Sprintf("Burnodd Scripts v%s", versionStr)

	// Generate new UUIDs
	manifest.Header.UUID = generateUUID()
	fmt.Printf("Header UUID: %s\n", manifest.Header.UUID)

	for i := range manifest.Modules {
		manifest.Modules[i].Version = manifest.Header.Version
		manifest.Modules[i].UUID = generateUUID()
		fmt.Printf("Module %d UUID: %s\n", i, manifest.Modules[i].UUID)
	}

	// Write back
	newData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return [3]int{}, fmt.Errorf("marshaling manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, append(newData, '\n'), 0644); err != nil {
		return [3]int{}, fmt.Errorf("writing manifest: %w", err)
	}

	return manifest.Header.Version, nil
}

func deleteOldPacks(outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Find and delete old .mcpack files
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("reading output directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mcpack") {
			path := filepath.Join(outputDir, entry.Name())
			fmt.Printf("Removing old pack: %s\n", entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", path, err)
			}
		}
	}

	return nil
}

func createMcpack(packDir, outputPath string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	// Create ZIP writer
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// Walk the pack directory and add files
	err = filepath.Walk(packDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path for the ZIP entry
		relPath, err := filepath.Rel(packDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		// Use forward slashes in ZIP (cross-platform)
		zipPath := strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

		// Only include specific file types for behavior packs
		ext := strings.ToLower(filepath.Ext(path))
		validExtensions := map[string]bool{
			".json":       true,
			".js":         true,
			".mcfunction": true,
			".lang":       true,
			".png":        true,
		}

		if !validExtensions[ext] {
			return nil
		}

		// Create ZIP entry
		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("creating zip entry %s: %w", zipPath, err)
		}

		// Open source file
		srcFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer srcFile.Close()

		// Copy content
		if _, err := io.Copy(writer, srcFile); err != nil {
			return fmt.Errorf("copying %s: %w", path, err)
		}

		fmt.Printf("  Added: %s\n", zipPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walking pack directory: %w", err)
	}

	return nil
}
