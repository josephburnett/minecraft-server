package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/realms"
)

// PackInstaller handles the installation of behavior packs to Realms.
type PackInstaller struct {
	client   *RealmsHTTPClient
	noBackup bool
}

// NewPackInstaller creates a new pack installer.
func NewPackInstaller(client *RealmsHTTPClient, noBackup bool) *PackInstaller {
	return &PackInstaller{
		client:   client,
		noBackup: noBackup,
	}
}

// Install downloads a Realm world, adds the behavior pack, and saves it for manual upload.
func (p *PackInstaller) Install(ctx context.Context, realm realms.Realm, packPath, packID string, version [3]int) error {
	packName := filepath.Base(packPath)

	fmt.Printf("Preparing behavior pack '%s' for Realm '%s'...\n", packName, realm.Name)

	// Step 1: Download the world
	fmt.Println("\n[1/3] Downloading world from Realm...")
	worldData, err := p.client.DownloadWorld(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to download world: %w", err)
	}
	fmt.Printf("Downloaded %d bytes\n", len(worldData))

	// Step 2: Create backup (unless disabled)
	if !p.noBackup {
		fmt.Println("\n[2/3] Creating backup...")
		if err := p.createBackup(realm, worldData); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	} else {
		fmt.Println("\n[2/3] Skipping backup (--no-backup)")
	}

	// Step 3: Modify the world and save to Minecraft worlds folder
	fmt.Println("\n[3/3] Injecting behavior pack...")
	outputPath, err := p.injectAndSave(worldData, packPath, packName, packID, version, realm.Name)
	if err != nil {
		return fmt.Errorf("failed to inject pack: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("SUCCESS! Modified world saved to Minecraft worlds folder")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\nWorld folder: %s\n", outputPath)
	fmt.Println("\nTo complete installation:")
	fmt.Println("1. Open Minecraft and go to your Realm settings")
	fmt.Println("2. Click the pencil icon to edit the Realm")
	fmt.Println("3. Select 'Replace World'")
	fmt.Printf("4. Choose '%s + Pack'\n", realm.Name)
	fmt.Println("5. Confirm and wait for the upload to complete")

	return nil
}

// createBackup saves a backup of the world data.
func (p *PackInstaller) createBackup(realm realms.Realm, data []byte) error {
	if err := os.MkdirAll("backups", 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	slot := realm.ActiveSlot
	if slot == 0 {
		slot = 1
	}
	filename := fmt.Sprintf("realm-%d-slot%d-%s.mcworld", realm.ID, slot, timestamp)
	backupPath := filepath.Join("backups", filename)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	fmt.Printf("Backup saved: %s\n", backupPath)
	return nil
}

// getMinecraftWorldsDir finds the Minecraft Bedrock worlds directory.
func getMinecraftWorldsDir() (string, error) {
	// Windows path via WSL
	home := os.Getenv("HOME")
	username := filepath.Base(home)

	// Try common Windows username locations
	candidates := []string{
		"/mnt/c/Users/" + username + "/AppData/Roaming/Minecraft Bedrock",
		"/mnt/c/Users/Joseph Burnett/AppData/Roaming/Minecraft Bedrock",
	}

	for _, base := range candidates {
		usersDir := filepath.Join(base, "Users")
		entries, err := os.ReadDir(usersDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "Shared" {
				worldsDir := filepath.Join(usersDir, entry.Name(), "games", "com.mojang", "minecraftWorlds")
				if _, err := os.Stat(worldsDir); err == nil {
					return worldsDir, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find Minecraft Bedrock worlds directory")
}

// injectAndSave adds the behavior pack to the world and saves to Minecraft worlds folder.
func (p *PackInstaller) injectAndSave(worldData []byte, packPath, packName, packID string, version [3]int, realmName string) (string, error) {
	// Open the world archive
	world, err := OpenMCWorld(worldData)
	if err != nil {
		return "", fmt.Errorf("failed to open world archive: %w", err)
	}

	// Check if pack source exists
	if _, err := os.Stat(packPath); err != nil {
		return "", fmt.Errorf("pack path not found: %s", packPath)
	}

	// Read pack manifest to get UUID and version
	manifestPath := filepath.Join(packPath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to read pack manifest: %w", err)
	}

	var manifest struct {
		Header struct {
			UUID    string `json:"uuid"`
			Version [3]int `json:"version"`
		} `json:"header"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse pack manifest: %w", err)
	}

	if packID == "" {
		packID = manifest.Header.UUID
	}
	if version == [3]int{0, 0, 0} {
		version = manifest.Header.Version
	}

	// Add the pack files to the world
	if err := world.AddBehaviorPack(packPath, packName); err != nil {
		return "", fmt.Errorf("failed to add pack files: %w", err)
	}

	// Update world_behavior_packs.json
	if err := world.AddOrUpdatePackRef(packID, version); err != nil {
		return "", fmt.Errorf("failed to update pack references: %w", err)
	}

	// Find Minecraft worlds directory
	worldsDir, err := getMinecraftWorldsDir()
	if err != nil {
		return "", err
	}

	// Create a new world folder with a unique name
	worldFolderName := fmt.Sprintf("%s + Pack", realmName)
	outputDir := filepath.Join(worldsDir, worldFolderName)

	// Remove existing folder if present
	os.RemoveAll(outputDir)

	// Create the world folder
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create world folder: %w", err)
	}

	// Write all files to the folder
	if err := world.ExtractTo(outputDir); err != nil {
		return "", fmt.Errorf("failed to extract world: %w", err)
	}

	// Update levelname.txt with the new name
	levelnameFile := filepath.Join(outputDir, "levelname.txt")
	if err := os.WriteFile(levelnameFile, []byte(worldFolderName), 0644); err != nil {
		return "", fmt.Errorf("failed to write levelname.txt: %w", err)
	}

	return outputDir, nil
}
