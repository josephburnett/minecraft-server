package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// BlockPlacement represents a single block to place in the world.
type BlockPlacement struct {
	X, Y, Z   int
	BlockName string
}

// ReadBlockFile reads a block placement file (CSV format: x,y,z,block_name).
func ReadBlockFile(path string) ([]BlockPlacement, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("block file not found: %s", path)
	}
	defer file.Close()

	var blocks []BlockPlacement
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ",", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("line %d: expected x,y,z,block_name, got %q", lineNum, line)
		}
		x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid x: %w", lineNum, err)
		}
		y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid y: %w", lineNum, err)
		}
		z, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid z: %w", lineNum, err)
		}
		blockName := strings.TrimSpace(parts[3])
		blocks = append(blocks, BlockPlacement{X: x, Y: y, Z: z, BlockName: blockName})
	}
	return blocks, scanner.Err()
}

// hotbarEntry tracks an item placed in the hotbar via creative inventory.
type hotbarEntry struct {
	slot              int32 // 0-8
	itemNetworkID     int32
	blockRuntimeID    int32
	stackNetworkID    int32
	creativeNetworkID uint32
}

// Builder handles block placement on a Realm connection.
type Builder struct {
	conn    *minecraft.Conn
	palette *Palette

	// hotbar maps block name to hotbar entry
	hotbar   map[string]*hotbarEntry
	hotbarMu sync.Mutex

	// nextSlot is the next hotbar slot to assign (0-8)
	nextSlot int32

	// requestID is an atomic counter for ItemStackRequest IDs
	requestID atomic.Int32

	// playerPos tracks the current player position
	playerPos mgl32.Vec3

	// tick tracks the current server tick
	tick atomic.Uint64

	// entityRuntimeID is the player's runtime ID
	entityRuntimeID uint64

	// placementDelay is the delay between block placements
	placementDelay time.Duration

	// moveDelay is the delay after teleporting
	moveDelay time.Duration

	// reachDistance is how far the player can reach to place blocks
	reachDistance float32

	// Stats
	placed  atomic.Int64
	failed  atomic.Int64
	skipped atomic.Int64
}

// NewBuilder creates a Builder for placing blocks on a connection.
func NewBuilder(conn *minecraft.Conn, palette *Palette) *Builder {
	gd := conn.GameData()
	b := &Builder{
		conn:            conn,
		palette:         palette,
		hotbar:          make(map[string]*hotbarEntry),
		playerPos:       gd.PlayerPosition,
		entityRuntimeID: gd.EntityRuntimeID,
		placementDelay:  80 * time.Millisecond,
		moveDelay:       100 * time.Millisecond,
		reachDistance:    5.0,
	}
	b.requestID.Store(1)
	return b
}

// newPlaceAction creates a PlaceStackRequestAction by setting exported fields
// on the embedded (unexported) transferStackRequestAction.
func newPlaceAction(count byte, src, dst protocol.StackRequestSlotInfo) *protocol.PlaceStackRequestAction {
	a := &protocol.PlaceStackRequestAction{}
	a.Count = count
	a.Source = src
	a.Destination = dst
	return a
}

// SetupHotbar places the required block types into the hotbar using creative inventory requests.
// It collects unique block types from the placement list and assigns them to hotbar slots.
func (b *Builder) SetupHotbar(blocks []BlockPlacement, responseCh <-chan *packet.ItemStackResponse) error {
	// Collect unique block types
	unique := make(map[string]bool)
	var blockTypes []string
	for _, bp := range blocks {
		if !unique[bp.BlockName] {
			unique[bp.BlockName] = true
			blockTypes = append(blockTypes, bp.BlockName)
		}
	}

	if len(blockTypes) > 9 {
		fmt.Printf("Warning: %d unique block types but only 9 hotbar slots. Will swap as needed.\n", len(blockTypes))
	}

	fmt.Printf("Setting up hotbar with %d block types...\n", len(blockTypes))

	// Place up to 9 block types in the hotbar
	limit := len(blockTypes)
	if limit > 9 {
		limit = 9
	}

	for i := 0; i < limit; i++ {
		blockName := blockTypes[i]
		if err := b.addToHotbar(blockName, int32(i), responseCh); err != nil {
			return fmt.Errorf("hotbar setup for %s: %w", blockName, err)
		}
	}

	fmt.Printf("Hotbar ready: %d block types loaded\n", len(b.hotbar))
	return nil
}

// addToHotbar requests a block from creative inventory into a hotbar slot.
func (b *Builder) addToHotbar(blockName string, slot int32, responseCh <-chan *packet.ItemStackResponse) error {
	// Look up the creative item for this block
	ci, ok := b.palette.CreativeItem(blockName)
	if !ok {
		return fmt.Errorf("block %q not found in creative inventory", blockName)
	}

	itemNetID, ok := b.palette.ItemNetworkID(blockName)
	if !ok {
		return fmt.Errorf("block %q not found in item registry", blockName)
	}

	reqID := b.requestID.Add(1)

	// Send ItemStackRequest with CraftCreative action followed by Place action.
	// This mimics what the client does: craft from creative, then place in hotbar.
	err := b.conn.WritePacket(&packet.ItemStackRequest{
		Requests: []protocol.ItemStackRequest{
			{
				RequestID: reqID,
				Actions: []protocol.StackRequestAction{
					// First: "craft" from creative inventory
					&protocol.CraftCreativeStackRequestAction{
						CreativeItemNetworkID: ci.CreativeItemNetworkID,
						NumberOfCrafts:        1,
					},
					// Second: place the crafted item into the hotbar slot
					newPlaceAction(
						1,
						protocol.StackRequestSlotInfo{
							Container: protocol.FullContainerName{
								ContainerID: protocol.ContainerCreatedOutput,
							},
							Slot:           50, // Creative output slot
							StackNetworkID: 0,
						},
						protocol.StackRequestSlotInfo{
							Container: protocol.FullContainerName{
								ContainerID: protocol.ContainerHotBar,
							},
							Slot:           byte(slot),
							StackNetworkID: 0, // empty slot
						},
					),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send ItemStackRequest: %w", err)
	}

	// Wait for response
	select {
	case resp := <-responseCh:
		for _, r := range resp.Responses {
			if r.RequestID == reqID {
				if r.Status != protocol.ItemStackResponseStatusOK {
					return fmt.Errorf("ItemStackRequest rejected: status=%d", r.Status)
				}
				// Extract the stack network ID from the response
				var stackNetID int32
				for _, containerInfo := range r.ContainerInfo {
					if containerInfo.Container.ContainerID == protocol.ContainerHotBar {
						for _, si := range containerInfo.SlotInfo {
							if si.Slot == byte(slot) {
								stackNetID = si.StackNetworkID
							}
						}
					}
				}

				b.hotbarMu.Lock()
				b.hotbar[blockName] = &hotbarEntry{
					slot:              slot,
					itemNetworkID:     int32(itemNetID),
					blockRuntimeID:    ci.Item.BlockRuntimeID,
					stackNetworkID:    stackNetID,
					creativeNetworkID: ci.CreativeItemNetworkID,
				}
				b.hotbarMu.Unlock()

				fmt.Printf("  Slot %d: %s (itemNetID=%d, blockRuntimeID=%d, stackNetID=%d)\n",
					slot, blockName, itemNetID, ci.Item.BlockRuntimeID, stackNetID)
				return nil
			}
		}
		return fmt.Errorf("no matching response for request %d", reqID)
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for ItemStackResponse")
	}
}

// getHotbarEntry returns the hotbar entry for a block name, swapping if needed.
func (b *Builder) getHotbarEntry(blockName string) (*hotbarEntry, error) {
	b.hotbarMu.Lock()
	entry, ok := b.hotbar[blockName]
	b.hotbarMu.Unlock()
	if ok {
		return entry, nil
	}
	return nil, fmt.Errorf("block %q not in hotbar (hotbar swap not yet implemented)", blockName)
}

// moveTo teleports the player to a position near the target block.
func (b *Builder) moveTo(target mgl32.Vec3) error {
	dist := target.Sub(b.playerPos).Len()
	if dist <= b.reachDistance {
		return nil // Already in range
	}

	// Teleport to a position near the target, with eye height offset
	b.playerPos = target.Add(mgl32.Vec3{0, 1.62, 0})

	err := b.conn.WritePacket(&packet.MovePlayer{
		EntityRuntimeID: b.entityRuntimeID,
		Position:        b.playerPos,
		Pitch:           90, // Looking down
		Yaw:             0,
		HeadYaw:         0,
		Mode:            packet.MoveModeTeleport,
		OnGround:        true,
		TeleportCause:   packet.TeleportCauseCommand,
		Tick:            b.tick.Load(),
	})
	if err != nil {
		return fmt.Errorf("move player: %w", err)
	}

	time.Sleep(b.moveDelay)
	return nil
}

// placeBlock places a single block at the given position.
func (b *Builder) placeBlock(bp BlockPlacement) error {
	entry, err := b.getHotbarEntry(bp.BlockName)
	if err != nil {
		return err
	}

	// Move player near the block if needed
	targetPos := mgl32.Vec3{float32(bp.X), float32(bp.Y), float32(bp.Z)}
	if err := b.moveTo(targetPos); err != nil {
		return err
	}

	// We place the block by clicking on the block below it (face = up = 1)
	adjacentPos := protocol.BlockPos{int32(bp.X), int32(bp.Y) - 1, int32(bp.Z)}
	face := int32(1) // Face up

	// Build the UseItemTransactionData for block placement
	txData := &protocol.UseItemTransactionData{
		ActionType:    protocol.UseItemActionClickBlock,
		TriggerType:   protocol.TriggerTypePlayerInput,
		BlockPosition: adjacentPos,
		BlockFace:     face,
		HotBarSlot:    entry.slot,
		HeldItem: protocol.ItemInstance{
			StackNetworkID: entry.stackNetworkID,
			Stack: protocol.ItemStack{
				ItemType: protocol.ItemType{
					NetworkID: entry.itemNetworkID,
				},
				BlockRuntimeID: entry.blockRuntimeID,
				Count:          1,
				HasNetworkID:   true,
			},
		},
		Position:         b.playerPos,
		ClickedPosition:  mgl32.Vec3{0.5, 1.0, 0.5}, // Center of top face
		BlockRuntimeID:   0,                           // Runtime ID of clicked block (0 = unknown/air)
		ClientPrediction: protocol.ClientPredictionSuccess,
	}

	err = b.conn.WritePacket(&packet.InventoryTransaction{
		LegacyRequestID: 0,
		TransactionData:  txData,
	})
	if err != nil {
		return fmt.Errorf("place block: %w", err)
	}

	return nil
}

// BuildStructure places all blocks from the placement list.
// It processes blocks in Y layers (bottom-up) for proper support.
func (b *Builder) BuildStructure(blocks []BlockPlacement, origin mgl32.Vec3) error {
	if len(blocks) == 0 {
		return fmt.Errorf("no blocks to place")
	}

	// Sort blocks bottom-up by Y, then by X, then by Z
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Y != blocks[j].Y {
			return blocks[i].Y < blocks[j].Y
		}
		if blocks[i].X != blocks[j].X {
			return blocks[i].X < blocks[j].X
		}
		return blocks[i].Z < blocks[j].Z
	})

	fmt.Printf("Building %d blocks at origin (%.0f, %.0f, %.0f)...\n",
		len(blocks), origin[0], origin[1], origin[2])

	startTime := time.Now()

	for i, bp := range blocks {
		// Offset by origin
		bp.X += int(origin[0])
		bp.Y += int(origin[1])
		bp.Z += int(origin[2])

		err := b.placeBlock(bp)
		if err != nil {
			b.failed.Add(1)
			if b.failed.Load() <= 10 {
				fmt.Printf("  Failed block %d (%d,%d,%d %s): %v\n",
					i, bp.X, bp.Y, bp.Z, bp.BlockName, err)
			}
			continue
		}
		b.placed.Add(1)

		// Progress reporting
		if (i+1)%100 == 0 || i+1 == len(blocks) {
			elapsed := time.Since(startTime)
			rate := float64(i+1) / elapsed.Seconds()
			fmt.Printf("  Progress: %d/%d (%.0f blocks/sec, %d failed)\n",
				i+1, len(blocks), rate, b.failed.Load())
		}

		time.Sleep(b.placementDelay)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Build complete: %d placed, %d failed, %d skipped in %v\n",
		b.placed.Load(), b.failed.Load(), b.skipped.Load(), elapsed.Round(time.Second))
	return nil
}

