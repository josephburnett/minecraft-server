package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PackRef represents a behavior pack reference in world_behavior_packs.json.
type PackRef struct {
	PackID  string `json:"pack_id"`
	Version [3]int `json:"version"`
}

// MCWorld represents a Minecraft world archive (.mcworld / .tar.gz format).
type MCWorld struct {
	files map[string][]byte
}

// OpenMCWorld opens a .mcworld archive from raw bytes.
// Supports both tar.gz (Realms) and zip formats.
func OpenMCWorld(data []byte) (*MCWorld, error) {
	// Check for gzip signature (0x1F 0x8B)
	if len(data) >= 2 && data[0] == 0x1F && data[1] == 0x8B {
		return openTarGz(data)
	}

	// Check for zip signature (PK)
	if len(data) >= 2 && data[0] == 0x50 && data[1] == 0x4B {
		return openZip(data)
	}

	return nil, fmt.Errorf("unknown archive format (first bytes: %x)", data[:min(4, len(data))])
}

// openTarGz opens a gzip-compressed tar archive.
func openTarGz(data []byte) (*MCWorld, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	world := &MCWorld{
		files: make(map[string][]byte),
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		if header.Typeflag == tar.TypeDir {
			continue
		}

		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", header.Name, err)
			}
			world.files[header.Name] = content
		}
	}

	return world, nil
}

// openZip opens a zip archive (legacy support).
func openZip(data []byte) (*MCWorld, error) {
	// Import archive/zip only when needed
	return nil, fmt.Errorf("zip format not implemented - Realms uses tar.gz")
}

// AddBehaviorPack adds a behavior pack from a source directory to the world.
// The pack will be added under behavior_packs/{packName}/.
func (w *MCWorld) AddBehaviorPack(srcPath string, packName string) error {
	// Read the manifest to verify it's a valid pack
	manifestPath := filepath.Join(srcPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("no manifest.json found in %s", srcPath)
	}

	// Walk the source directory and add all files
	basePath := "behavior_packs/" + packName + "/"

	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Normalize path separators to forward slashes
		zipPath := basePath + strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
		w.files[zipPath] = content

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to add behavior pack: %w", err)
	}

	return nil
}

// GetWorldBehaviorPacks reads the current world_behavior_packs.json.
func (w *MCWorld) GetWorldBehaviorPacks() ([]PackRef, error) {
	data, exists := w.files["world_behavior_packs.json"]
	if !exists {
		return []PackRef{}, nil
	}

	var packs []PackRef
	if err := json.Unmarshal(data, &packs); err != nil {
		return nil, fmt.Errorf("failed to parse world_behavior_packs.json: %w", err)
	}

	return packs, nil
}

// UpdateWorldBehaviorPacksJSON updates the world_behavior_packs.json with the given packs.
func (w *MCWorld) UpdateWorldBehaviorPacksJSON(packs []PackRef) error {
	data, err := json.MarshalIndent(packs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal packs: %w", err)
	}

	w.files["world_behavior_packs.json"] = data
	return nil
}

// AddOrUpdatePackRef adds a pack reference or updates it if it already exists.
func (w *MCWorld) AddOrUpdatePackRef(packID string, version [3]int) error {
	packs, err := w.GetWorldBehaviorPacks()
	if err != nil {
		return err
	}

	// Check if pack already exists and update it
	found := false
	for i, pack := range packs {
		if pack.PackID == packID {
			packs[i].Version = version
			found = true
			break
		}
	}

	// Add if not found
	if !found {
		packs = append(packs, PackRef{
			PackID:  packID,
			Version: version,
		})
	}

	return w.UpdateWorldBehaviorPacksJSON(packs)
}

// Save serializes the world back to tar.gz format.
func (w *MCWorld) Save() ([]byte, error) {
	buf := new(bytes.Buffer)
	gzWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzWriter)

	for name, content := range w.files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write header for %s: %w", name, err)
		}

		if _, err := tarWriter.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write content for %s: %w", name, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// SaveAsZip serializes the world to .mcworld (zip) format.
func (w *MCWorld) SaveAsZip() ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for name, content := range w.files {
		f, err := zipWriter.Create(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create zip entry for %s: %w", name, err)
		}

		if _, err := f.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write content for %s: %w", name, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// ExtractTo extracts all world files to a directory.
// It flattens the Realms "worlds/world/" prefix to match local world structure.
func (w *MCWorld) ExtractTo(dir string) error {
	for name, content := range w.files {
		// Flatten Realms structure: worlds/world/* -> *
		flatName := name
		if strings.HasPrefix(name, "worlds/world/") {
			flatName = strings.TrimPrefix(name, "worlds/world/")
		} else if strings.HasPrefix(name, "worlds/") {
			// Skip other worlds/ paths that aren't world data
			continue
		}

		filePath := filepath.Join(dir, flatName)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", name, err)
		}

		// Write file
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	return nil
}

// ListFiles returns all file paths in the world.
func (w *MCWorld) ListFiles() []string {
	files := make([]string, 0, len(w.files))
	for name := range w.files {
		files = append(files, name)
	}
	return files
}

// HasFile checks if a file exists in the world.
func (w *MCWorld) HasFile(path string) bool {
	_, exists := w.files[path]
	return exists
}
