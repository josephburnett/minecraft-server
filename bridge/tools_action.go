package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
