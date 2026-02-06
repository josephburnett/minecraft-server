.PHONY: auth mcp pack test

# Authenticate with Xbox Live (run once, opens browser)
auth: bridge/bridge
	bridge/bridge -auth

# Start MCP server (normally started by Claude Code via .mcp.json)
mcp: bridge/bridge
	bridge/bridge

# Build .mcpack for Realm import
pack: pack-builder/pack-builder
	pack-builder/pack-builder

# Run behavior pack tests
test:
	npx vitest run

bridge/bridge: bridge/*.go
	cd bridge && go build -o bridge .

pack-builder/pack-builder: pack-builder/*.go
	cd pack-builder && go build -o pack-builder .
