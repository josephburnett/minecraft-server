.PHONY: start stop restart logs reload status upload upload-local play install-pack pack ping
.PHONY: maze sphere cube pyramid test

# Docker commands
start:
	docker compose up -d

stop:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

reload:
	docker exec minecraft-bedrock send-command "reload"

status:
	docker compose ps

play:
	powershell.exe -Command "Start-Process 'minecraft://connect/?serverUrl=127.0.0.1&serverPort=19133'"

# Structure generators - write chunks to file
# Usage: make maze [WIDTH=15] [HEIGHT=7] [LENGTH=15] [BLOCK=minecraft:stone_bricks]
maze:
	@node tools/generators/maze.js $(or $(WIDTH),15) $(or $(HEIGHT),7) $(or $(LENGTH),15) $(or $(BLOCK),minecraft:stone_bricks) > structure.chunks

# Usage: make sphere [RADIUS=5] [BLOCK=minecraft:glass] [HOLLOW=true]
sphere:
	@node tools/generators/sphere.js $(or $(RADIUS),5) $(or $(BLOCK),minecraft:glass) $(or $(HOLLOW),true) > structure.chunks

# Usage: make cube [SIZE=10] [BLOCK=minecraft:stone] [HOLLOW=true]
cube:
	@node tools/generators/cube.js $(or $(SIZE),10) $(or $(BLOCK),minecraft:stone) $(or $(HOLLOW),true) > structure.chunks

# Usage: make pyramid [BASE=15] [BLOCK=minecraft:sandstone]
pyramid:
	@node tools/generators/pyramid.js $(or $(BASE),15) $(or $(BLOCK),minecraft:sandstone) > structure.chunks

# Usage: make test [PATTERN=frame] [SIZE=10]
# Patterns: checkerboard, stripes, frame, cross, corner, small, line
test:
	@node tools/generators/test.js $(or $(PATTERN),frame) $(or $(SIZE),10) > structure.chunks

# Upload chunks to Realm via gophertunnel (set REALM_INVITE or create .realm-invite)
upload: tools/upload-realm/realmctl
	tools/upload-realm/realmctl -chunks structure.chunks

# Install behavior pack to Realm (one-time setup)
install-pack: tools/upload-realm/realmctl
	tools/upload-realm/realmctl -install-pack

# Ping mode: connect to Realm and send periodic time queries
ping: tools/upload-realm/realmctl
	tools/upload-realm/realmctl -ping

tools/upload-realm/realmctl: tools/upload-realm/*.go
	cd tools/upload-realm && go build -o realmctl .

# Upload chunks to local Docker server
upload-local:
	@while read chunk; do \
		docker exec minecraft-bedrock send-command "scriptevent burnodd:chunk $$chunk"; \
	done < structure.chunks

# Build .mcpack file for import into Minecraft
# Output: output/burnodd_scripts.mcpack (double-click to import)
pack: tools/pack-builder/pack-builder
	tools/pack-builder/pack-builder

tools/pack-builder/pack-builder: tools/pack-builder/*.go
	cd tools/pack-builder && go build -o pack-builder .
