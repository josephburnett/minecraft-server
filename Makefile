.PHONY: start stop restart logs reload status
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

# Structure generators
# Usage: make maze [WIDTH=15] [HEIGHT=7] [LENGTH=15] [BLOCK=minecraft:stone_bricks]
maze:
	@node tools/generators/maze.js $(or $(WIDTH),15) $(or $(HEIGHT),7) $(or $(LENGTH),15) $(or $(BLOCK),minecraft:stone_bricks)

# Usage: make sphere [RADIUS=5] [BLOCK=minecraft:glass] [HOLLOW=true]
sphere:
	@node tools/generators/sphere.js $(or $(RADIUS),5) $(or $(BLOCK),minecraft:glass) $(or $(HOLLOW),true)

# Usage: make cube [SIZE=10] [BLOCK=minecraft:stone] [HOLLOW=true]
cube:
	@node tools/generators/cube.js $(or $(SIZE),10) $(or $(BLOCK),minecraft:stone) $(or $(HOLLOW),true)

# Usage: make pyramid [BASE=15] [BLOCK=minecraft:sandstone]
pyramid:
	@node tools/generators/pyramid.js $(or $(BASE),15) $(or $(BLOCK),minecraft:sandstone)

# Usage: make test [PATTERN=frame] [SIZE=10]
# Patterns: checkerboard, stripes, frame, cross, corner, small, line
test:
	@node tools/generators/test.js $(or $(PATTERN),frame) $(or $(SIZE),10)

# Generate and send to server
# Usage: make build-maze [WIDTH=15] [HEIGHT=7] [LENGTH=15] [BLOCK=minecraft:stone_bricks]
build-maze:
	@CMD=$$(node tools/generators/maze.js $(or $(WIDTH),15) $(or $(HEIGHT),7) $(or $(LENGTH),15) $(or $(BLOCK),minecraft:stone_bricks)) && \
	docker exec minecraft-bedrock send-command "$$CMD"

build-sphere:
	@CMD=$$(node tools/generators/sphere.js $(or $(RADIUS),5) $(or $(BLOCK),minecraft:glass) $(or $(HOLLOW),true)) && \
	docker exec minecraft-bedrock send-command "$$CMD"

build-cube:
	@CMD=$$(node tools/generators/cube.js $(or $(SIZE),10) $(or $(BLOCK),minecraft:stone) $(or $(HOLLOW),true)) && \
	docker exec minecraft-bedrock send-command "$$CMD"

build-pyramid:
	@CMD=$$(node tools/generators/pyramid.js $(or $(BASE),15) $(or $(BLOCK),minecraft:sandstone)) && \
	docker exec minecraft-bedrock send-command "$$CMD"

build-test:
	@CMD=$$(node tools/generators/test.js $(or $(PATTERN),frame) $(or $(SIZE),10)) && \
	docker exec minecraft-bedrock send-command "$$CMD"
