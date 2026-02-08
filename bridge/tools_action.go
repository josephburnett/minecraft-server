package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func registerActionTools(s *server.MCPServer, state *GameState) {
	// chat
	s.AddTool(
		mcp.NewTool("chat",
			mcp.WithDescription("Send a chat message to the Realm as the connected player"),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("The chat message to send"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			msg, err := req.RequireString("message")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, xuid := state.Identity()
			conn := state.ServerConn()
			if conn == nil {
				return mcp.NewToolResultError("server connection not available"), nil
			}
			if err := conn.WritePacket(&packet.Text{
				TextType:   packet.TextTypeChat,
				SourceName: name,
				XUID:       xuid,
				Message:    msg,
			}); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("send error: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("sent chat: %s", msg)), nil
		},
	)

	// command
	s.AddTool(
		mcp.NewTool("command",
			mcp.WithDescription("Execute a Minecraft command on the Realm (e.g. 'time set day', 'give @s diamond 64'). Do not include the leading slash."),
			mcp.WithString("command",
				mcp.Required(),
				mcp.Description("The command to execute (without leading /)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			cmd, err := req.RequireString("command")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			// Ensure leading slash
			cmd = strings.TrimPrefix(cmd, "/")

			// Send as chat message — CommandRequest packets can cause disconnects on Realms
			name, xuid := state.Identity()
			conn := state.ServerConn()
			if conn == nil {
				return mcp.NewToolResultError("server connection not available"), nil
			}
			if err := conn.WritePacket(&packet.Text{
				TextType:   packet.TextTypeChat,
				SourceName: name,
				XUID:       xuid,
				Message:    "/" + cmd,
			}); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("command error: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("executed: /%s", cmd)), nil
		},
	)

	// teleport
	s.AddTool(
		mcp.NewTool("teleport",
			mcp.WithDescription("Teleport the player to specific coordinates"),
			mcp.WithNumber("x", mcp.Required(), mcp.Description("X coordinate")),
			mcp.WithNumber("y", mcp.Required(), mcp.Description("Y coordinate")),
			mcp.WithNumber("z", mcp.Required(), mcp.Description("Z coordinate")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			x, err := req.RequireFloat("x")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			y, err := req.RequireFloat("y")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			z, err := req.RequireFloat("z")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			msg := fmt.Sprintf("/tp @s %.2f %.2f %.2f", x, y, z)
			name, xuid := state.Identity()
			conn := state.ServerConn()
			if conn == nil {
				return mcp.NewToolResultError("server connection not available"), nil
			}
			if err := conn.WritePacket(&packet.Text{
				TextType:   packet.TextTypeChat,
				SourceName: name,
				XUID:       xuid,
				Message:    msg,
			}); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("teleport error: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("teleporting to (%.2f, %.2f, %.2f)", x, y, z)), nil
		},
	)

	// toggle_packet_logging
	s.AddTool(
		mcp.NewTool("toggle_packet_logging",
			mcp.WithDescription("Enable or disable verbose logging of building-related packets (block placement, breaking, inventory, etc.)"),
			mcp.WithBoolean("enabled",
				mcp.Required(),
				mcp.Description("Whether to enable verbose packet logging"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			enabled, err := req.RequireBool("enabled")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			state.SetVerbosePacketLog(enabled)
			status := "disabled"
			if enabled {
				status = "enabled"
			}
			slog.Info("packet logging toggled", "enabled", enabled)
			return mcp.NewToolResultText(fmt.Sprintf("verbose packet logging %s", status)), nil
		},
	)

	// place_blocks
	s.AddTool(
		mcp.NewTool("place_blocks",
			mcp.WithDescription("Place blocks in the world by sending the full client placement packet sequence. Requires creative mode or the blocks in inventory. Each entry specifies coordinates and a block name."),
			mcp.WithString("blocks",
				mcp.Required(),
				mcp.Description(`JSON array of block placements, e.g. [{"x":0,"y":64,"z":0,"block_name":"minecraft:stone"}]`),
			),
			mcp.WithNumber("delay_ms",
				mcp.Description("Delay in milliseconds between placements (default 100)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			blocksJSON, err := req.RequireString("blocks")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			delayMs := req.GetInt("delay_ms", 100)
			delay := time.Duration(delayMs) * time.Millisecond

			var blocks []struct {
				X         int    `json:"x"`
				Y         int    `json:"y"`
				Z         int    `json:"z"`
				BlockName string `json:"block_name"`
			}
			if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid blocks JSON: %v", err)), nil
			}
			if len(blocks) == 0 {
				return mcp.NewToolResultError("blocks array is empty"), nil
			}

			conn := state.ServerConn()
			if conn == nil {
				return mcp.NewToolResultError("server connection not available"), nil
			}

			placed := 0
			for i, b := range blocks {
				select {
				case <-ctx.Done():
					return mcp.NewToolResultText(fmt.Sprintf("interrupted after %d/%d blocks", placed, len(blocks))), nil
				default:
				}

				if err := placeBlock(conn, state, int32(b.X), int32(b.Y), int32(b.Z), b.BlockName); err != nil {
					slog.Warn("place_blocks: placement failed", "index", i, "block", b.BlockName, "error", err)
					return mcp.NewToolResultError(fmt.Sprintf("failed at block %d (%s at %d,%d,%d): %v", i, b.BlockName, b.X, b.Y, b.Z, err)), nil
				}
				placed++

				if delay > 0 && i < len(blocks)-1 {
					time.Sleep(delay)
				}
			}

			return mcp.NewToolResultText(fmt.Sprintf("placed %d blocks", placed)), nil
		},
	)

	// upload_structure
	s.AddTool(
		mcp.NewTool("upload_structure",
			mcp.WithDescription("Upload a .chunks structure file to the Realm. Each line is sent as a '!chunk' chat message to the behavior pack script. This is a long-running operation."),
			mcp.WithString("file",
				mcp.Required(),
				mcp.Description("Path to the .chunks file to upload"),
			),
			mcp.WithNumber("delay_ms",
				mcp.Description("Delay in milliseconds between chunk sends (default 50)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			filePath, err := req.RequireString("file")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			delayMs := req.GetInt("delay_ms", 50)
			delay := time.Duration(delayMs) * time.Millisecond

			// Read chunks file
			chunks, err := readChunksFile(filePath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if len(chunks) == 0 {
				return mcp.NewToolResultError("no chunks found in file"), nil
			}

			name, xuid := state.Identity()
			conn := state.ServerConn()
			if conn == nil {
				return mcp.NewToolResultError("server connection not available"), nil
			}

			slog.Info("uploading structure", "file", filePath, "chunks", len(chunks), "delay_ms", delayMs)

			for i, chunk := range chunks {
				// Check for cancellation between sends
				select {
				case <-ctx.Done():
					return mcp.NewToolResultText(fmt.Sprintf("interrupted after %d/%d chunks", i, len(chunks))), nil
				default:
				}

				if err := conn.WritePacket(&packet.Text{
					TextType:   packet.TextTypeChat,
					SourceName: name,
					XUID:       xuid,
					Message:    fmt.Sprintf("!chunk %s", chunk),
				}); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("send error at chunk %d: %v", i+1, err)), nil
				}

				if delay > 0 {
					time.Sleep(delay)
				}
			}

			return mcp.NewToolResultText(fmt.Sprintf("uploaded %d chunks from %s", len(chunks), filePath)), nil
		},
	)
}

// readChunksFile reads a line-delimited chunks file, skipping empty lines.
func readChunksFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open chunks file: %w", err)
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

// placeBlock sends the 4-packet block placement sequence to the server connection,
// mimicking what the real client sends when a player places a block.
func placeBlock(conn *minecraft.Conn, state *GameState, x, y, z int32, blockName string) error {
	networkID, ok := state.ResolveItemNetworkID(blockName)
	if !ok {
		return fmt.Errorf("unknown block name %q (not in item registry)", blockName)
	}

	entityID := state.EntityID()
	posX, posY, posZ, _, _, _ := state.Position()

	targetPos := protocol.BlockPos{x, y - 1, z} // block below — we "click on top"
	newPos := protocol.BlockPos{x, y, z}         // where the block will appear

	heldItem := protocol.ItemInstance{
		StackNetworkID: 0,
		Stack: protocol.ItemStack{
			ItemType: protocol.ItemType{
				NetworkID:     networkID,
				MetadataValue: 0,
			},
			BlockRuntimeID: 0,
			Count:          1,
			HasNetworkID:   false,
		},
	}

	// 1. PlayerAction(StartItemUseOn)
	if err := conn.WritePacket(&packet.PlayerAction{
		EntityRuntimeID: entityID,
		ActionType:      protocol.PlayerActionStartItemUseOn,
		BlockPosition:   targetPos,
		ResultPosition:  newPos,
		BlockFace:       1, // Up
	}); err != nil {
		return fmt.Errorf("StartItemUseOn: %w", err)
	}

	// 2. InventoryTransaction(ClickBlock)
	if err := conn.WritePacket(&packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType:     protocol.UseItemActionClickBlock,
			TriggerType:    protocol.TriggerTypePlayerInput,
			BlockPosition:  targetPos,
			BlockFace:      1, // Up
			HotBarSlot:     0,
			HeldItem:       heldItem,
			Position:       mgl32.Vec3{posX, posY, posZ},
			ClickedPosition: mgl32.Vec3{0.5, 0.5, 0.5},
			BlockRuntimeID: 0,
			ClientPrediction: protocol.ClientPredictionSuccess,
		},
	}); err != nil {
		return fmt.Errorf("ClickBlock: %w", err)
	}

	// 3. InventoryTransaction(ClickAir)
	if err := conn.WritePacket(&packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType:     protocol.UseItemActionClickAir,
			TriggerType:    protocol.TriggerTypePlayerInput,
			BlockPosition:  protocol.BlockPos{0, 0, 0},
			BlockFace:      -1,
			HotBarSlot:     0,
			HeldItem:       heldItem,
			Position:       mgl32.Vec3{posX, posY, posZ},
			ClickedPosition: mgl32.Vec3{0, 0, 0},
			BlockRuntimeID: 0,
			ClientPrediction: protocol.ClientPredictionSuccess,
		},
	}); err != nil {
		return fmt.Errorf("ClickAir: %w", err)
	}

	// 4. PlayerAction(StopItemUseOn)
	if err := conn.WritePacket(&packet.PlayerAction{
		EntityRuntimeID: entityID,
		ActionType:      protocol.PlayerActionStopItemUseOn,
		BlockPosition:   newPos,
		ResultPosition:  protocol.BlockPos{0, 0, 0},
		BlockFace:       0, // Down
	}); err != nil {
		return fmt.Errorf("StopItemUseOn: %w", err)
	}

	return nil
}
