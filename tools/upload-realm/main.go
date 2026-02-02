package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"golang.org/x/oauth2"
)

const tokenFile = ".realm-token"
const defaultPackPath = "bedrock/behavior_packs/burnodd_scripts"

func main() {
	// Flags for both modes
	installPack := flag.Bool("install-pack", false, "Install behavior pack to Realm")
	packPath := flag.String("pack-path", defaultPackPath, "Path to behavior pack folder")
	noBackup := flag.Bool("no-backup", false, "Skip creating backup before pack installation")

	// Flags for chunk uploader mode
	chunksFile := flag.String("chunks", "structure.chunks", "Path to chunks file")

	// Flags for ping mode
	ping := flag.Bool("ping", false, "Connect to Realm and send periodic time queries to test connection")
	duration := flag.Int("duration", 60, "Seconds to stay connected in ping mode (0 = until Ctrl+C)")

	flag.Parse()

	if *installPack {
		if err := runPackInstaller(*packPath, *noBackup); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *ping {
		if err := runPing(*duration); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := runChunkUploader(*chunksFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runPing(duration int) error {
	inviteCode, err := getRealmInvite()
	if err != nil {
		return err
	}

	tokenSource, err := getTokenSource()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up OS signal listener for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, shutting down...")
		cancel()
	}()

	// If duration > 0, set a timer to cancel
	if duration > 0 {
		time.AfterFunc(time.Duration(duration)*time.Second, func() {
			fmt.Printf("\nDuration (%ds) expired, shutting down...\n", duration)
			cancel()
		})
	}

	realmsClient := realms.NewClient(tokenSource, nil)
	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		return fmt.Errorf("realm lookup error: %w", err)
	}

	address, err := realm.Address(ctx)
	if err != nil {
		return fmt.Errorf("realm address error: %w", err)
	}

	fmt.Printf("Connecting to %s (%s)...\n", realm.Name, address)

	dialer := minecraft.Dialer{
		TokenSource: tokenSource,
	}

	conn, err := dialer.Dial("raknet", address)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	if err := conn.DoSpawn(); err != nil {
		return fmt.Errorf("spawn error: %w", err)
	}

	// Log GameData summary
	gd := conn.GameData()
	fmt.Printf("Spawned! World: %s, Gamemode: %d, Position: (%.1f, %.1f, %.1f)\n",
		gd.WorldName, gd.PlayerGameMode,
		gd.PlayerPosition.X(), gd.PlayerPosition.Y(), gd.PlayerPosition.Z())

	if duration > 0 {
		fmt.Printf("Ping mode active for %ds (Ctrl+C to stop early)...\n", duration)
	} else {
		fmt.Println("Ping mode active (Ctrl+C to stop)...")
	}

	startTime := time.Now()
	var totalPackets atomic.Int64
	var lastTime atomic.Int32
	// Test chat message size limits
	testSizes := []int{256, 512, 1024, 2048}
	go func() {
		time.Sleep(2 * time.Second)
		for _, size := range testSizes {
			if ctx.Err() != nil {
				return
			}
			msg := fmt.Sprintf("[%d] ", size)
			for len(msg) < size {
				msg += "A"
			}
			msg = msg[:size]
			err := conn.WritePacket(&packet.Text{
				TextType: packet.TextTypeChat,
				Message:  msg,
			})
			if err != nil {
				fmt.Printf("[send] %d chars: error: %v\n", size, err)
				return
			}
			fmt.Printf("[send] sent %d chars\n", size)
			time.Sleep(2 * time.Second)
		}
		fmt.Println("[send] all size tests complete")
	}()

	// Main goroutine: read loop
	packetCounts := make(map[string]int)
	lastSummary := time.Now()
	lastTimePrint := time.Now()

	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			// Check if context was cancelled (clean shutdown)
			if ctx.Err() != nil {
				break
			}
			fmt.Printf("[read] error: %v\n", err)
			break
		}
		totalPackets.Add(1)

		switch p := pk.(type) {
		case *packet.SetTime:
			lastTime.Store(int32(p.Time))
			if time.Since(lastTimePrint) >= 2*time.Second {
				fmt.Printf("[time] %d (tick)\n", p.Time)
				lastTimePrint = time.Now()
			}
		case *packet.Text:
			fmt.Printf("[chat] (%d) %s: %s\n", p.TextType, p.SourceName, p.Message)
		case *packet.PacketViolationWarning:
			fmt.Printf("[violation] severity=%d type=%d packetID=%d: %s\n",
				p.Severity, p.Type, p.PacketID, p.ViolationContext)
		case *packet.Disconnect:
			fmt.Printf("[disconnect] %s\n", p.Message)
			cancel()
		default:
			name := fmt.Sprintf("%T", pk)
			packetCounts[name]++
		}

		// Summarize packet counts every 10 seconds
		if time.Since(lastSummary) >= 10*time.Second && len(packetCounts) > 0 {
			fmt.Printf("[summary] packets in last 10s:")
			for name, count := range packetCounts {
				fmt.Printf(" %s=%d", name, count)
			}
			fmt.Println()
			packetCounts = make(map[string]int)
			lastSummary = time.Now()
		}

		if ctx.Err() != nil {
			break
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Session ended. Duration: %s, Packets received: %d\n",
		elapsed.Round(time.Second), totalPackets.Load())
	return nil
}

func runPackInstaller(packPath string, noBackup bool) error {
	// Get realm invite code
	inviteCode, err := getRealmInvite()
	if err != nil {
		return err
	}

	// Authenticate
	tokenSource, err := getTokenSource()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	ctx := context.Background()

	// Get Realm info
	realmsClient := realms.NewClient(tokenSource, nil)
	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		return fmt.Errorf("realm lookup error: %w", err)
	}

	fmt.Printf("Found Realm: %s (ID: %d)\n", realm.Name, realm.ID)

	// Create HTTP client and installer
	httpClient := NewRealmsHTTPClient(tokenSource)
	installer := NewPackInstaller(httpClient, noBackup)

	// Install the pack
	// Version will be read from the pack's manifest.json
	return installer.Install(ctx, realm, packPath, "", [3]int{})
}

func runChunkUploader(chunksFile string) error {
	// Get realm invite code
	inviteCode, err := getRealmInvite()
	if err != nil {
		return err
	}

	// Read chunks
	chunks, err := readChunks(chunksFile)
	if err != nil {
		return err
	}

	fmt.Printf("Uploading %d chunks to Realm...\n", len(chunks))

	// Authenticate (with token caching)
	tokenSource, err := getTokenSource()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	// Get Realm address
	ctx := context.Background()
	realmsClient := realms.NewClient(tokenSource, nil)

	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		return fmt.Errorf("realm lookup error: %w", err)
	}

	address, err := realm.Address(ctx)
	if err != nil {
		return fmt.Errorf("realm address error: %w", err)
	}

	fmt.Printf("Connecting to %s (%s)...\n", realm.Name, address)

	// Connect
	dialer := minecraft.Dialer{
		TokenSource: tokenSource,
	}

	conn, err := dialer.Dial("raknet", address)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	// Spawn
	if err := conn.DoSpawn(); err != nil {
		return fmt.Errorf("spawn error: %w", err)
	}

	fmt.Println("Connected! Sending chunks as chat messages...")

	// Send chunks as chat messages
	for i, chunk := range chunks {
		err := conn.WritePacket(&packet.Text{
			TextType: packet.TextTypeChat,
			Message:  fmt.Sprintf("!chunk %s", chunk),
		})
		if err != nil {
			return fmt.Errorf("send error: %w", err)
		}

		if (i+1)%50 == 0 {
			fmt.Printf("Progress: %d / %d\n", i+1, len(chunks))
		}

		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("Done! Sent %d chunks.\n", len(chunks))
	time.Sleep(time.Second)
	return nil
}

func getTokenSource() (oauth2.TokenSource, error) {
	// Try to load cached token
	token, err := loadToken()
	if err == nil {
		fmt.Println("Using cached authentication...")
		return auth.RefreshTokenSource(token), nil
	}

	// Request new token
	fmt.Println("Authenticating (check browser)...")
	token, err = auth.RequestLiveToken()
	if err != nil {
		return nil, err
	}

	// Save token for next time
	if err := saveToken(token); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not cache token: %v\n", err)
	}

	return auth.RefreshTokenSource(token), nil
}

func loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func saveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(tokenFile, data, 0600)
}

func getRealmInvite() (string, error) {
	// Check environment variable
	if code := os.Getenv("REALM_INVITE"); code != "" {
		return code, nil
	}

	// Check .realm-invite file
	exe, _ := os.Executable()
	dir := filepath.Dir(filepath.Dir(exe))
	inviteFile := filepath.Join(dir, ".realm-invite")

	// Also check current working directory
	if _, err := os.Stat(inviteFile); os.IsNotExist(err) {
		inviteFile = ".realm-invite"
	}

	data, err := os.ReadFile(inviteFile)
	if err != nil {
		return "", fmt.Errorf("no realm invite found; set REALM_INVITE or create .realm-invite file")
	}

	return strings.TrimSpace(string(data)), nil
}

func readChunks(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("chunks file not found: %s", path)
	}
	defer file.Close()

	var chunks []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			chunks = append(chunks, line)
		}
	}

	return chunks, scanner.Err()
}
