package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"golang.org/x/oauth2"
)

const tokenFile = ".realm-token"
const defaultPackPath = "bedrock/behavior_packs/burnodd_scripts"

func main() {
	// Flags for all modes
	installPack := flag.Bool("install-pack", false, "Install behavior pack to Realm")
	packPath := flag.String("pack-path", defaultPackPath, "Path to behavior pack folder")
	noBackup := flag.Bool("no-backup", false, "Skip creating backup before pack installation")
	ping := flag.Bool("ping", false, "Connect to Realm, send a test command, stay connected briefly")
	build := flag.Bool("build", false, "Build structure on Realm by placing blocks directly")

	// Flags for chunk uploader mode
	chunksFile := flag.String("chunks", "structure.chunks", "Path to chunks file")

	// Flags for build mode
	blocksFile := flag.String("blocks", "structure.blocks", "Path to blocks file (CSV: x,y,z,block_name)")

	flag.Parse()

	var err error
	if *installPack {
		err = runPackInstaller(*packPath, *noBackup)
	} else if *ping {
		err = runPing()
	} else if *build {
		err = runBuild(*blocksFile)
	} else {
		err = runChunkUploader(*chunksFile)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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

// runPing connects to the Realm, sends a test command, and stays connected
// for 30 seconds to verify the connection stays alive with a read loop.
func runPing() error {
	inviteCode, err := getRealmInvite()
	if err != nil {
		return err
	}

	tokenSource, err := getTokenSource()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	ctx := context.Background()
	realmsClient := realms.NewClient(tokenSource, nil)

	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		return fmt.Errorf("realm lookup error: %w", err)
	}

	address, network, err := resolveRealmConnection(ctx, tokenSource, &realm)
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s (%s) via %s...\n", realm.Name, address, network)

	dialer := minecraft.Dialer{
		TokenSource:       tokenSource,
		EnableClientCache: false,
		PacketFunc: func(header packet.Header, payload []byte, src, dst net.Addr) {
			// Log raw bytes for CommandRequest (ID 77) packets
			if header.PacketID == packet.IDCommandRequest {
				fmt.Printf("[PacketFunc] CommandRequest raw payload (%d bytes), src=%s dst=%s\n", len(payload), src, dst)
				fmt.Printf("[PacketFunc] Header: PacketID=%d SenderSub=%d TargetSub=%d\n",
					header.PacketID, header.SenderSubClient, header.TargetSubClient)
				for i, v := range payload {
					fmt.Printf("%02x ", v)
					if (i+1)%16 == 0 {
						fmt.Println()
					}
				}
				if len(payload)%16 != 0 {
					fmt.Println()
				}
			}
		},
	}

	conn, err := dialer.Dial(network, address)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	if err := conn.DoSpawn(); err != nil {
		return fmt.Errorf("spawn error: %w", err)
	}

	gd := conn.GameData()
	gameModes := map[int32]string{0: "survival", 1: "creative", 2: "adventure", 3: "survival_spectator", 4: "creative_spectator", 5: "default"}
	modeName := gameModes[gd.PlayerGameMode]
	if modeName == "" {
		modeName = fmt.Sprintf("unknown(%d)", gd.PlayerGameMode)
	}
	fmt.Printf("Game data: version=%s world=%s gamemode=%s\n", gd.BaseGameVersion, gd.WorldName, modeName)
	fmt.Printf("Player position: (%.1f, %.1f, %.1f)\n", gd.PlayerPosition[0], gd.PlayerPosition[1], gd.PlayerPosition[2])
	fmt.Printf("Server authoritative inventory: %v\n", gd.ServerAuthoritativeInventory)
	fmt.Printf("Items in registry: %d\n", len(gd.Items))
	fmt.Println("Spawned! Starting 30-second ping test...")

	// First, just read packets for 5 seconds without sending anything
	// to confirm the connection is stable.
	go func() {
		time.Sleep(5 * time.Second)

		// Send a simple /help command first to test basic command execution
		fmt.Println("[5s] Sending /help command...")
		if err := conn.WritePacket(&packet.CommandRequest{
			CommandLine: "/help",
			CommandOrigin: protocol.CommandOrigin{
				Origin: protocol.CommandOriginPlayer,
				UUID:   uuid.New(),
			},
		}); err != nil {
			fmt.Printf("Command error: %v\n", err)
			return
		}
		fmt.Println("[5s] /help sent successfully")

		time.Sleep(3 * time.Second)

		// Then try scriptevent
		fmt.Println("[8s] Sending /scriptevent test...")
		if err := conn.WritePacket(&packet.CommandRequest{
			CommandLine: "/scriptevent burnodd:chunk test:0:1:dGVzdA==",
			CommandOrigin: protocol.CommandOrigin{
				Origin: protocol.CommandOriginPlayer,
				UUID:   uuid.New(),
			},
		}); err != nil {
			fmt.Printf("Command error: %v\n", err)
		}
		fmt.Println("[8s] /scriptevent sent successfully")
	}()

	// Close connection after 30 seconds to break the read loop
	go func() {
		time.Sleep(30 * time.Second)
		fmt.Println("Ping timeout reached, closing connection...")
		conn.Close()
	}()

	// Read loop — keeps connection alive, logs ALL packets
	start := time.Now()
	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			fmt.Printf("[%.1fs] Read loop ended: %v\n", time.Since(start).Seconds(), err)
			return nil
		}
		elapsed := time.Since(start).Seconds()
		switch p := pk.(type) {
		case *packet.Text:
			fmt.Printf("[%.1fs] [Chat] type=%d message=%q\n", elapsed, p.TextType, p.Message)
		case *packet.ScriptMessage:
			fmt.Printf("[%.1fs] [ScriptMessage] id=%s data=%s\n", elapsed, p.Identifier, string(p.Data))
		case *packet.CommandOutput:
			if p.SuccessCount > 0 {
				fmt.Printf("[%.1fs] [CommandOutput] success (count=%d)\n", elapsed, p.SuccessCount)
			} else {
				fmt.Printf("[%.1fs] [CommandOutput] failed: %v\n", elapsed, p.OutputMessages)
			}
		case *packet.PacketViolationWarning:
			fmt.Printf("[%.1fs] [ViolationWarning] type=%d severity=%d packetID=%d context=%s\n",
				elapsed, p.Type, p.Severity, p.PacketID, p.ViolationContext)
		case *packet.Disconnect:
			fmt.Printf("[%.1fs] [Disconnect] reason=%d message=%s\n", elapsed, p.Reason, p.Message)
			return nil
		default:
			fmt.Printf("[%.1fs] [Packet] %T\n", elapsed, pk)
		}
	}
}

// runBuild connects to the Realm and places blocks directly using
// InventoryTransaction packets (creative mode block placement).
func runBuild(blocksFile string) error {
	// Read block placements
	blocks, err := ReadBlockFile(blocksFile)
	if err != nil {
		return err
	}
	fmt.Printf("Loaded %d block placements from %s\n", len(blocks), blocksFile)

	inviteCode, err := getRealmInvite()
	if err != nil {
		return err
	}

	tokenSource, err := getTokenSource()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	ctx := context.Background()
	realmsClient := realms.NewClient(tokenSource, nil)

	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		return fmt.Errorf("realm lookup error: %w", err)
	}

	address, network, err := resolveRealmConnection(ctx, tokenSource, &realm)
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s (%s) via %s...\n", realm.Name, address, network)

	dialer := minecraft.Dialer{
		TokenSource:       tokenSource,
		EnableClientCache: false,
	}

	conn, err := dialer.Dial(network, address)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	if err := conn.DoSpawn(); err != nil {
		return fmt.Errorf("spawn error: %w", err)
	}

	gd := conn.GameData()
	gameModes := map[int32]string{0: "survival", 1: "creative", 2: "adventure"}
	modeName := gameModes[gd.PlayerGameMode]
	if modeName == "" {
		modeName = fmt.Sprintf("unknown(%d)", gd.PlayerGameMode)
	}
	fmt.Printf("Connected! gamemode=%s pos=(%.1f, %.1f, %.1f)\n",
		modeName, gd.PlayerPosition[0], gd.PlayerPosition[1], gd.PlayerPosition[2])

	if gd.PlayerGameMode != 1 {
		fmt.Println("Warning: Player is not in creative mode. Block placement may fail.")
	}

	// Build palette from game data
	palette := NewPalette(gd.Items)

	// Channel for ItemStackResponse packets
	responseCh := make(chan *packet.ItemStackResponse, 10)

	// Start read loop in background to keep connection alive and capture responses
	readDone := make(chan error, 1)
	go func() {
		for {
			pk, err := conn.ReadPacket()
			if err != nil {
				readDone <- err
				return
			}
			switch p := pk.(type) {
			case *packet.CreativeContent:
				fmt.Printf("Received CreativeContent: %d groups, %d items\n",
					len(p.Groups), len(p.Items))
				palette.LoadCreativeContent(p)
			case *packet.ItemStackResponse:
				responseCh <- p
			case *packet.PacketViolationWarning:
				fmt.Printf("[ViolationWarning] type=%d severity=%d packetID=%d context=%s\n",
					p.Type, p.Severity, p.PacketID, p.ViolationContext)
			case *packet.Disconnect:
				fmt.Printf("[Disconnect] reason=%d message=%s\n", p.Reason, p.Message)
				readDone <- fmt.Errorf("disconnected: %s", p.Message)
				return
			case *packet.Text:
				fmt.Printf("[Chat] %s\n", p.Message)
			}
		}
	}()

	// Wait a moment for CreativeContent to arrive
	fmt.Println("Waiting for creative inventory data...")
	time.Sleep(3 * time.Second)

	palette.DumpStats()

	// Create builder and set up hotbar
	builder := NewBuilder(conn, palette)

	if err := builder.SetupHotbar(blocks, responseCh); err != nil {
		return fmt.Errorf("hotbar setup: %w", err)
	}

	// Build the structure at the player's current position
	origin := gd.PlayerPosition
	// Place blocks relative to ground level (player feet, not eye height)
	origin[1] = origin[1] - 1.62

	if err := builder.BuildStructure(blocks, origin); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	// Give time for final packets
	time.Sleep(2 * time.Second)
	return nil
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

	// Get Realm address with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	realmsClient := realms.NewClient(tokenSource, nil)

	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		return fmt.Errorf("realm lookup error: %w", err)
	}

	address, network, err := resolveRealmConnection(ctx, tokenSource, &realm)
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s (%s) via %s...\n", realm.Name, address, network)

	// Connect with client cache enabled
	dialer := minecraft.Dialer{
		TokenSource:       tokenSource,
		EnableClientCache: true,
	}

	conn, err := dialer.Dial(network, address)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	// Spawn
	if err := conn.DoSpawn(); err != nil {
		return fmt.Errorf("spawn error: %w", err)
	}

	// Check for burnodd_scripts pack by name
	packFound := false
	for _, pack := range conn.ResourcePacks() {
		if strings.Contains(strings.ToLower(pack.Name()), "burnodd") {
			packFound = true
			fmt.Printf("Found behavior pack: %s\n", pack.Name())
			break
		}
	}
	if !packFound {
		fmt.Println("Warning: burnodd_scripts behavior pack not detected in ResourcePacks.")
		fmt.Println("This may be normal — Realms may not report behavior packs here.")
		fmt.Println("Continuing anyway. If commands fail, run 'make install-pack' first.")
	}

	fmt.Println("Connected! Sending commands...")

	// Track command results
	var successCount atomic.Int64
	var failCount atomic.Int64

	// Send commands in a goroutine; main goroutine runs the read loop
	var sendErr error
	go func() {
		time.Sleep(2 * time.Second) // let connection stabilize

		for i, chunk := range chunks {
			cmd := fmt.Sprintf("/scriptevent burnodd:chunk %s", chunk)
			err := conn.WritePacket(&packet.CommandRequest{
				CommandLine: cmd,
				CommandOrigin: protocol.CommandOrigin{
					Origin: protocol.CommandOriginPlayer,
					UUID:   uuid.New(),
				},
			})
			if err != nil {
				sendErr = fmt.Errorf("command error at chunk %d: %w", i, err)
				conn.Close()
				return
			}

			if (i+1)%50 == 0 {
				fmt.Printf("Progress: %d / %d (ack: %d ok, %d fail)\n",
					i+1, len(chunks), successCount.Load(), failCount.Load())
			}

			time.Sleep(50 * time.Millisecond)
		}

		fmt.Printf("All %d commands sent. Waiting for final responses...\n", len(chunks))
		time.Sleep(3 * time.Second)
		conn.Close() // break the read loop
	}()

	// Read loop — keeps connection alive and processes responses
	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			// Connection closed (either by us after sending, or by server)
			if sendErr != nil {
				return sendErr
			}
			break
		}
		switch p := pk.(type) {
		case *packet.Text:
			fmt.Printf("[Chat] %s\n", p.Message)
		case *packet.CommandOutput:
			if p.SuccessCount > 0 {
				successCount.Add(1)
			} else {
				failCount.Add(1)
				if len(p.OutputMessages) > 0 {
					fmt.Printf("[Command Failed] %v\n", p.OutputMessages)
				}
			}
		case *packet.PacketViolationWarning:
			fmt.Printf("[Violation Warning] type=%d severity=%d packetID=%d context=%s\n",
				p.Type, p.Severity, p.PacketID, p.ViolationContext)
		case *packet.Disconnect:
			fmt.Printf("[Disconnect] %s\n", p.Message)
			if sendErr != nil {
				return sendErr
			}
			return fmt.Errorf("disconnected by server: %s", p.Message)
		}
	}

	// Summary
	fmt.Printf("Done! Sent %d commands. Results: %d succeeded, %d failed.\n",
		len(chunks), successCount.Load(), failCount.Load())
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

// ensurePort appends the default Bedrock port if the address has no port.
// Realms API sometimes returns addresses without a port.
func ensurePort(address string) string {
	if _, _, err := net.SplitHostPort(address); err != nil {
		return net.JoinHostPort(address, "19132")
	}
	return address
}

// isNetherNetAddress returns true if the address looks like a NetherNet network
// ID (a UUID or numeric string) rather than a traditional IP:port address.
// The Realms API returns a UUID when the realm uses the NETHERNET protocol.
func isNetherNetAddress(address string) bool {
	// Remove port if present
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	// Check if it looks like a UUID (contains dashes, no dots)
	if strings.Contains(host, "-") && !strings.Contains(host, ".") {
		return true
	}
	// Check if it parses as a valid IP address
	if net.ParseIP(host) != nil {
		return false
	}
	// If it doesn't parse as an IP and doesn't contain dots (no hostname), it's nethernet
	if !strings.Contains(host, ".") {
		return true
	}
	return false
}

// setupNetherNet authenticates with PlayFab to get an MCToken for NetherNet
// signaling, and configures the nethernet network layer.
func setupNetherNet(tokenSource oauth2.TokenSource) error {
	fmt.Println("Setting up NetherNet authentication (PlayFab)...")
	pf, err := NewPlayFabClient(tokenSource)
	if err != nil {
		return fmt.Errorf("PlayFab auth: %w", err)
	}
	SetNetherNetToken(pf.MCToken())
	fmt.Println("NetherNet signaling authentication ready.")
	return nil
}

// resolveRealmConnection resolves a Realm address and returns the address and
// network type ("raknet" or "nethernet") to use for dialing.
func resolveRealmConnection(ctx context.Context, tokenSource oauth2.TokenSource, realm *realms.Realm) (address, network string, err error) {
	addrCtx, addrCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer addrCancel()
	address, err = realm.Address(addrCtx)
	if err != nil {
		return "", "", fmt.Errorf("realm address error: %w", err)
	}
	fmt.Printf("Raw realm address: %q\n", address)

	if isNetherNetAddress(address) {
		fmt.Println("Detected NetherNet protocol (WebRTC)")

		// Resolve the MPSD session handle UUID to a numeric WebRTC network ID
		fmt.Println("Resolving MPSD session handle to WebRTC network ID...")
		networkID, err := ResolveNetherNetID(ctx, tokenSource, address)
		if err != nil {
			return "", "", fmt.Errorf("resolve nethernet network ID: %w", err)
		}
		fmt.Printf("Resolved WebRTC network ID: %s\n", networkID)

		if err := setupNetherNet(tokenSource); err != nil {
			return "", "", err
		}
		return networkID, "nethernet", nil
	}

	address = ensurePort(address)
	return address, "raknet", nil
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
