# Minecraft Realm Bridge

MCP server + Bedrock proxy for controlling a Minecraft Realm from Claude Code.

## Setup

1. Create `.realm-invite` with your Realm invite code
2. Authenticate with Xbox Live (opens browser, run once):
   ```
   make auth
   ```
3. `.mcp.json` is already configured — restart Claude Code and the bridge starts automatically

## Behavior Pack

The `behavior_pack/` directory contains scripts that run on the Realm. Build and import via Minecraft:

```
make pack
```

This creates a `.mcpack` file in `output/` that can be imported by double-clicking.

## Testing

```
make test
```

## File Structure

```
minecraft-server/
├── .mcp.json             # MCP server config (auto-starts bridge)
├── Makefile              # auth, mcp, pack, test
├── bridge/               # MCP server + Bedrock proxy (Go)
├── pack-builder/         # .mcpack builder (Go)
├── behavior_pack/        # Realm behavior pack (JS + mcfunctions)
├── tests/                # Vitest tests for behavior pack
└── output/               # Built .mcpack files
```
