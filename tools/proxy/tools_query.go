package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerQueryTools(s *server.MCPServer, state *GameState) {
	// get_status
	s.AddTool(
		mcp.NewTool("get_status",
			mcp.WithDescription("Get the current proxy connection status, player name, and whether the realm is connected"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, _ := state.Identity()
			result := map[string]any{
				"status":          state.Status(),
				"player_name":     name,
				"realm_connected": state.Status() == StatusConnected,
			}
			return jsonResult(result)
		},
	)

	// get_position
	s.AddTool(
		mcp.NewTool("get_position",
			mcp.WithDescription("Get the player's current position, rotation, and dimension in the Minecraft world"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			x, y, z, pitch, yaw, dim := state.Position()
			result := map[string]any{
				"x":         x,
				"y":         y,
				"z":         z,
				"pitch":     pitch,
				"yaw":       yaw,
				"dimension": dimensionName(dim),
			}
			return jsonResult(result)
		},
	)

	// get_inventory
	s.AddTool(
		mcp.NewTool("get_inventory",
			mcp.WithDescription("Get the player's current inventory contents (non-empty slots)"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			items := state.Inventory()
			return jsonResult(items)
		},
	)

	// get_players
	s.AddTool(
		mcp.NewTool("get_players",
			mcp.WithDescription("Get the list of players currently online in the Realm"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			players := state.Players()
			return jsonResult(players)
		},
	)

	// get_chat_history
	s.AddTool(
		mcp.NewTool("get_chat_history",
			mcp.WithDescription("Get recent chat messages from the Realm. Returns up to the last 100 messages."),
			mcp.WithNumber("count",
				mcp.Description("Number of recent messages to return (default 20, max 100)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			count := req.GetInt("count", 20)
			if count > maxChatHistory {
				count = maxChatHistory
			}
			messages := state.ChatHistory(count)
			return jsonResult(messages)
		},
	)

	// get_world_info
	s.AddTool(
		mcp.NewTool("get_world_info",
			mcp.WithDescription("Get world information including name, time, game mode, health, and spawn position"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := requireConnected(state); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			worldName, worldTime, gameMode, health, spawnPos := state.WorldInfo()
			result := map[string]any{
				"world_name": worldName,
				"time":       worldTime,
				"game_mode":  gameModeName(gameMode),
				"health":     health,
				"spawn_pos": map[string]int{
					"x": int(spawnPos.X()),
					"y": int(spawnPos.Y()),
					"z": int(spawnPos.Z()),
				},
			}
			return jsonResult(result)
		},
	)
}

func requireConnected(state *GameState) error {
	if state.Status() != StatusConnected {
		return fmt.Errorf("not connected to realm (status: %s)", state.Status())
	}
	return nil
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(data)), nil
}

func dimensionName(dim int32) string {
	switch dim {
	case 0:
		return "overworld"
	case 1:
		return "nether"
	case 2:
		return "the_end"
	default:
		return fmt.Sprintf("unknown(%d)", dim)
	}
}

func gameModeName(mode int32) string {
	switch mode {
	case 0:
		return "survival"
	case 1:
		return "creative"
	case 2:
		return "adventure"
	case 3:
		return "spectator"
	default:
		return fmt.Sprintf("unknown(%d)", mode)
	}
}
