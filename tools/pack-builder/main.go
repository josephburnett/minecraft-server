// Package main creates a .mcpack file from the behavior pack directory.
// An .mcpack is a ZIP file that can be double-clicked to import into Minecraft.
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	packDir := flag.String("pack", "bedrock/behavior_packs/burnodd_scripts", "Path to behavior pack directory")
	output := flag.String("output", "output/burnodd_scripts.mcpack", "Output .mcpack file path")
	flag.Parse()

	if err := createMcpack(*packDir, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s\n", *output)
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
