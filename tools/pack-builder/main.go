// Package main creates a .mcpack file from the behavior pack directory.
// An .mcpack is a ZIP file that can be double-clicked to import into Minecraft.
package main

import (
	"archive/zip"
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
	output := flag.String("output", "output/burnodd_scripts.mcpack", "Output .mcpack file path")
	noBump := flag.Bool("no-bump", false, "Skip version bump")
	flag.Parse()

	if !*noBump {
		newVersion, err := bumpVersion(*packDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error bumping version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Version: %d.%d.%d\n", newVersion[0], newVersion[1], newVersion[2])
	}

	if err := createMcpack(*packDir, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s\n", *output)
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

	// Also bump module versions to match
	for i := range manifest.Modules {
		manifest.Modules[i].Version = manifest.Header.Version
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
