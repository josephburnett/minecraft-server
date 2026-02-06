import { world, system } from "@minecraft/server";
import { buildBlocksAt } from "./structure-builder.js";
import { consumeMarker } from "./marker.js";
import { createGrid, gridToBlocks, getPlayerRotation } from "./shapes.js";

/**
 * Shuffle array in place (Fisher-Yates)
 * @param {Array} array
 * @returns {Array}
 */
export function shuffle(array) {
    for (let i = array.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [array[i], array[j]] = [array[j], array[i]];
    }
    return array;
}

/**
 * Generate a maze grid using growing tree algorithm with post-processing
 * @param {number} width - X dimension (will be made odd)
 * @param {number} height - Y dimension (will be made odd)
 * @param {number} length - Z dimension (will be made odd)
 * @returns {{grid: boolean[][][], size: number[], entrance: number[], exit: number[]}}
 */
export function generateMazeGrid(width, height, length) {
    // Ensure odd dimensions for proper maze walls
    const mazeW = width % 2 === 0 ? width + 1 : width;
    const mazeH = height % 2 === 0 ? height + 1 : height;
    const mazeL = length % 2 === 0 ? length + 1 : length;

    // Cell dimensions (maze cells are 2x2 with walls between)
    const cellsX = Math.floor(mazeW / 2);
    const cellsZ = Math.floor(mazeL / 2);

    // Initialize grid - true = wall/block, false = air
    const grid = createGrid(mazeW, mazeH, mazeL, true);

    // Track visited cells
    const visited = [];
    for (let x = 0; x < cellsX; x++) {
        visited[x] = [];
        for (let z = 0; z < cellsZ; z++) {
            visited[x][z] = false;
        }
    }

    // Directions: [dx, dz]
    const directions = [
        [0, 1],   // North (+Z)
        [0, -1],  // South (-Z)
        [1, 0],   // East (+X)
        [-1, 0]   // West (-X)
    ];

    // Convert cell coordinates to grid coordinates
    function cellToGrid(cx, cz) {
        return [cx * 2 + 1, cz * 2 + 1];
    }

    // Carve passage at grid position (all heights except floor and ceiling)
    function carveCell(gx, gz) {
        for (let y = 1; y < mazeH - 1; y++) {
            grid[gx][y][gz] = false;
        }
    }

    // Carve passage between two cells (removes the wall between them)
    function carvePassage(cx1, cz1, cx2, cz2) {
        const [gx1, gz1] = cellToGrid(cx1, cz1);
        const [gx2, gz2] = cellToGrid(cx2, cz2);
        const wallX = (gx1 + gx2) / 2;
        const wallZ = (gz1 + gz2) / 2;

        for (let y = 1; y < mazeH - 1; y++) {
            grid[wallX][y][wallZ] = false;
        }
    }

    // Growing tree maze generation (75% newest / 25% random)
    function generateMaze(startCx, startCz) {
        visited[startCx][startCz] = true;
        const [sgx, sgz] = cellToGrid(startCx, startCz);
        carveCell(sgx, sgz);

        const active = [[startCx, startCz]];

        while (active.length > 0) {
            // Pick cell: 75% newest, 25% random
            const idx = Math.random() < 0.75
                ? active.length - 1
                : Math.floor(Math.random() * active.length);
            const [cx, cz] = active[idx];

            // Find unvisited neighbors
            const unvisited = [];
            for (const [dx, dz] of directions) {
                const nx = cx + dx;
                const nz = cz + dz;
                if (nx >= 0 && nx < cellsX && nz >= 0 && nz < cellsZ && !visited[nx][nz]) {
                    unvisited.push([nx, nz]);
                }
            }

            if (unvisited.length > 0) {
                const pick = unvisited[Math.floor(Math.random() * unvisited.length)];
                const [nx, nz] = pick;
                visited[nx][nz] = true;
                const [ngx, ngz] = cellToGrid(nx, nz);
                carveCell(ngx, ngz);
                carvePassage(cx, cz, nx, nz);
                active.push([nx, nz]);
            } else {
                active.splice(idx, 1);
            }
        }
    }

    // Generate the base maze starting from corner
    generateMaze(0, 0);

    // Create entrance at the front (z=0 side)
    for (let y = 1; y < mazeH - 1; y++) {
        grid[1][y][0] = false;
    }

    // Create exit at the back (z=max side)
    for (let y = 1; y < mazeH - 1; y++) {
        grid[mazeW - 2][y][mazeL - 1] = false;
    }

    // --- Post-processing ---

    // Check if a cell-space grid position is carved (air)
    function isCarved(gx, gz) {
        return gx >= 0 && gx < mazeW && gz >= 0 && gz < mazeL && !grid[gx][1][gz];
    }

    // 1. Extend dead ends — 40% chance to carve one cell deeper
    for (let cx = 0; cx < cellsX; cx++) {
        for (let cz = 0; cz < cellsZ; cz++) {
            const [gx, gz] = cellToGrid(cx, cz);
            if (!isCarved(gx, gz)) continue;

            // A dead end in cell space: only 1 carved neighbor among the
            // wall positions (the passages between cells)
            let openPassages = 0;
            const blockedDirs = [];
            for (const [dx, dz] of directions) {
                const wallGx = gx + dx;
                const wallGz = gz + dz;
                if (isCarved(wallGx, wallGz)) {
                    openPassages++;
                } else {
                    // Could extend in this direction if neighbor cell exists
                    const ncx = cx + dx;
                    const ncz = cz + dz;
                    if (ncx >= 0 && ncx < cellsX && ncz >= 0 && ncz < cellsZ) {
                        blockedDirs.push([dx, dz]);
                    }
                }
            }

            if (openPassages === 1 && blockedDirs.length > 0 && Math.random() < 0.4) {
                const [dx, dz] = blockedDirs[Math.floor(Math.random() * blockedDirs.length)];
                carvePassage(cx, cz, cx + dx, cz + dz);
            }
        }
    }

    // 2. Compute solution path via BFS from entrance to exit (grid coords)
    const entrance = [1, 0];  // gx, gz of entrance opening
    const exit = [mazeW - 2, mazeL - 1];

    const solutionSet = new Set();
    {
        const prev = {};
        const queue = [entrance.join(",")];
        const visitedBFS = new Set(queue);

        while (queue.length > 0) {
            const key = queue.shift();
            const [sx, sz] = key.split(",").map(Number);

            if (sx === exit[0] && sz === exit[1]) {
                // Trace back and record solution cells
                let k = key;
                while (k) {
                    solutionSet.add(k);
                    k = prev[k];
                }
                break;
            }

            for (const [dx, dz] of directions) {
                const nx = sx + dx;
                const nz = sz + dz;
                const nk = nx + "," + nz;
                if (!visitedBFS.has(nk) && isCarved(nx, nz)) {
                    visitedBFS.add(nk);
                    prev[nk] = key;
                    queue.push(nk);
                }
            }
        }
    }

    function isOnSolution(gx, gz) {
        return solutionSet.has(gx + "," + gz);
    }

    // 3. Add loops (~20% of internal walls between carved cells removed)
    // Walls that separate two carved cells sit at odd+even or even+odd grid positions
    for (let gx = 1; gx < mazeW - 1; gx++) {
        for (let gz = 1; gz < mazeL - 1; gz++) {
            if (!grid[gx][1][gz]) continue; // already carved

            // Check if this wall separates two carved cells
            // Horizontal wall (between cells differing in X)
            if (gx % 2 === 0 && gz % 2 === 1) {
                if (isCarved(gx - 1, gz) && isCarved(gx + 1, gz)) {
                    // Bias away from solution: 5% if either side on solution, 20% otherwise
                    const chance = (isOnSolution(gx - 1, gz) || isOnSolution(gx + 1, gz)) ? 0.05 : 0.20;
                    if (Math.random() < chance) {
                        carveCell(gx, gz);
                    }
                }
            }
            // Vertical wall (between cells differing in Z)
            if (gx % 2 === 1 && gz % 2 === 0) {
                if (isCarved(gx, gz - 1) && isCarved(gx, gz + 1)) {
                    const chance = (isOnSolution(gx, gz - 1) || isOnSolution(gx, gz + 1)) ? 0.05 : 0.20;
                    if (Math.random() < chance) {
                        carveCell(gx, gz);
                    }
                }
            }
        }
    }

    // 4. Create deceptive rooms — small 2x2 or 3x3 cell clusters biased toward exit half
    const roomAttempts = Math.floor(cellsX * cellsZ * 0.05);
    for (let i = 0; i < roomAttempts; i++) {
        const roomSize = Math.random() < 0.5 ? 2 : 3;
        // Bias toward exit half (higher Z)
        const rcx = Math.floor(Math.random() * (cellsX - roomSize + 1));
        const halfZ = Math.floor(cellsZ / 2);
        const rcz = halfZ + Math.floor(Math.random() * (cellsZ - roomSize + 1 - halfZ));
        if (rcz + roomSize > cellsZ) continue;

        // Remove all internal walls and pillars within this cell cluster
        const [roomGxMin] = cellToGrid(rcx, rcz);
        const [roomGxMax] = cellToGrid(rcx + roomSize - 1, rcz);
        const [, roomGzMin] = cellToGrid(rcx, rcz);
        const [, roomGzMax] = cellToGrid(rcx, rcz + roomSize - 1);
        for (let gx = roomGxMin; gx <= roomGxMax; gx++) {
            for (let gz = roomGzMin; gz <= roomGzMax; gz++) {
                carveCell(gx, gz);
            }
        }
    }

    return {
        grid,
        size: [mazeW, mazeH, mazeL],
        entrance: [1, 1, 0],
        exit: [mazeW - 2, 1, mazeL - 1]
    };
}

/**
 * Rotate blocks around the Y axis
 * @param {Array<[number, number, number, string]>} blocks
 * @param {number[]} size - [width, height, length]
 * @param {number} rotation - 0, 90, 180, or 270 degrees
 * @returns {Array<[number, number, number, string]>}
 */
export function rotateBlocks(blocks, size, rotation) {
    if (rotation === 0) return blocks;

    const [w, h, l] = size;

    return blocks.map(([x, y, z, blockType]) => {
        let nx, nz;
        switch (rotation) {
            case 90:  // 90° clockwise
                nx = l - 1 - z;
                nz = x;
                break;
            case 180: // 180°
                nx = w - 1 - x;
                nz = l - 1 - z;
                break;
            case 270: // 270° clockwise (90° counter-clockwise)
                nx = z;
                nz = w - 1 - x;
                break;
            default:
                nx = x;
                nz = z;
        }
        return [nx, y, nz, blockType];
    });
}

/**
 * Build a maze at the marker location (if set) or player's location
 * @param {Player} player
 * @param {object} options
 * @param {number} [options.width=15] - Maze width
 * @param {number} [options.height=7] - Maze height
 * @param {number} [options.length=15] - Maze length
 * @param {string} [options.block="minecraft:stone_bricks"] - Block type
 */
export function buildMaze(player, options = {}) {
    const width = options.width || 15;
    const height = options.height || 7;
    const length = options.length || 15;
    const blockType = options.block || "minecraft:stone_bricks";

    // Try to consume marker FIRST (removes blocks before building)
    const marker = consumeMarker();
    let buildX, buildY, buildZ, rotation, dimension;

    if (marker) {
        buildX = marker.x;
        buildY = marker.y;
        buildZ = marker.z;
        rotation = marker.rotation;
        dimension = marker.dimension;
        player.sendMessage(`§aBuilding ${width}x${height}x${length} maze at marker...`);
    } else {
        // Fall back to player position and facing
        const pos = player.location;
        buildX = Math.floor(pos.x);
        buildY = Math.floor(pos.y);
        buildZ = Math.floor(pos.z);
        rotation = getPlayerRotation(player);
        dimension = player.dimension;
        player.sendMessage(`§aGenerating ${width}x${height}x${length} maze at player...`);
    }

    // Generate the maze
    const { grid, size } = generateMazeGrid(width, height, length);
    const [mazeW, mazeH, mazeL] = size;

    // Convert to blocks
    let blocks = gridToBlocks(grid, blockType);

    // Rotate maze based on marker/player rotation
    blocks = rotateBlocks(blocks, size, rotation);

    // Effective dimensions after rotation (90°/270° swap width and length)
    const effectiveW = (rotation === 90 || rotation === 270) ? mazeL : mazeW;
    const effectiveL = (rotation === 90 || rotation === 270) ? mazeW : mazeL;

    // Center horizontally and place below the reference point
    // Ceiling ends up at buildY - 1, floor at buildY - mazeH
    buildX -= Math.floor(effectiveW / 2);
    buildZ -= Math.floor(effectiveL / 2);
    buildY -= mazeH;

    const totalBlocks = blocks.length;
    player.sendMessage(`§7Placing ${totalBlocks} blocks...`);

    // Build at the target location
    buildBlocksAt(dimension, buildX, buildY, buildZ, blocks, (placed) => {
        player.sendMessage(`§a§lMaze complete! §r§7${placed} blocks placed`);
        player.sendMessage(`§7Tip: Exit is at the opposite corner`);
    });
}

/**
 * Parse maze options from a message string
 * Format: "width height length block" (all optional)
 * @param {string} message
 * @returns {object}
 */
export function parseMazeOptions(message) {
    const options = {};
    const parts = message.trim().split(/\s+/);

    if (parts[0] && !isNaN(parseInt(parts[0]))) {
        options.width = parseInt(parts[0]);
    }
    if (parts[1] && !isNaN(parseInt(parts[1]))) {
        options.height = parseInt(parts[1]);
    }
    if (parts[2] && !isNaN(parseInt(parts[2]))) {
        options.length = parseInt(parts[2]);
    }
    if (parts[3]) {
        options.block = parts[3].includes(":") ? parts[3] : `minecraft:${parts[3]}`;
    }

    return options;
}

/**
 * Initialize maze scriptevent handler
 */
export function initMazeHandler() {
    system.afterEvents.scriptEventReceive.subscribe((event) => {
        if (event.id === "burnodd:maze") {
            const player = event.sourceEntity;
            if (!player) {
                world.sendMessage("§cMaze generation requires a player source");
                return;
            }

            try {
                const options = parseMazeOptions(event.message || "");
                buildMaze(player, options);
            } catch (e) {
                player.sendMessage(`§cMaze generation failed: ${e.message}`);
            }
        }
    });
}
