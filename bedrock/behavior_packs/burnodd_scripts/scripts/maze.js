import { world, system } from "@minecraft/server";
import { buildBlocksAt } from "./structure-builder.js";
import { getMarker, clearMarker } from "./marker.js";

/**
 * Create a 3D boolean grid initialized to a value
 * @param {number} width
 * @param {number} height
 * @param {number} length
 * @param {boolean} value - initial value
 * @returns {boolean[][][]}
 */
export function createGrid(width, height, length, value = false) {
    const grid = [];
    for (let x = 0; x < width; x++) {
        grid[x] = [];
        for (let y = 0; y < height; y++) {
            grid[x][y] = [];
            for (let z = 0; z < length; z++) {
                grid[x][y][z] = value;
            }
        }
    }
    return grid;
}

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
 * Generate a maze grid using recursive backtracking
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

    // Carve passage between two cells
    function carvePassage(cx1, cz1, cx2, cz2) {
        const [gx1, gz1] = cellToGrid(cx1, cz1);
        const [gx2, gz2] = cellToGrid(cx2, cz2);
        const wallX = (gx1 + gx2) / 2;
        const wallZ = (gz1 + gz2) / 2;

        for (let y = 1; y < mazeH - 1; y++) {
            grid[wallX][y][wallZ] = false;
        }
    }

    // Recursive backtracking maze generation (iterative to avoid stack overflow)
    function generateMaze(startCx, startCz) {
        const stack = [[startCx, startCz]];

        while (stack.length > 0) {
            const [cx, cz] = stack[stack.length - 1];

            if (!visited[cx][cz]) {
                visited[cx][cz] = true;
                const [gx, gz] = cellToGrid(cx, cz);
                carveCell(gx, gz);
            }

            // Find unvisited neighbors
            const unvisited = [];
            const dirs = shuffle([...directions]);

            for (const [dx, dz] of dirs) {
                const nx = cx + dx;
                const nz = cz + dz;

                if (nx >= 0 && nx < cellsX && nz >= 0 && nz < cellsZ && !visited[nx][nz]) {
                    unvisited.push([nx, nz, dx, dz]);
                }
            }

            if (unvisited.length > 0) {
                // Pick a random unvisited neighbor
                const [nx, nz] = unvisited[0];
                carvePassage(cx, cz, nx, nz);
                stack.push([nx, nz]);
            } else {
                // Backtrack
                stack.pop();
            }
        }
    }

    // Generate the maze starting from corner
    generateMaze(0, 0);

    // Create entrance at the front (z=0 side)
    for (let y = 1; y < mazeH - 1; y++) {
        grid[1][y][0] = false;
    }

    // Create exit at the back (z=max side)
    for (let y = 1; y < mazeH - 1; y++) {
        grid[mazeW - 2][y][mazeL - 1] = false;
    }

    return {
        grid,
        size: [mazeW, mazeH, mazeL],
        entrance: [1, 1, 0],
        exit: [mazeW - 2, 1, mazeL - 1]
    };
}

/**
 * Convert maze grid to block array
 * @param {boolean[][][]} grid - 3D boolean grid (true = block)
 * @param {string} blockType - Block type ID
 * @returns {Array<[number, number, number, string]>}
 */
export function gridToBlocks(grid, blockType) {
    const blocks = [];
    const width = grid.length;
    const height = grid[0]?.length || 0;
    const length = grid[0]?.[0]?.length || 0;

    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                if (grid[x][y][z]) {
                    blocks.push([x, y, z, blockType]);
                }
            }
        }
    }

    return blocks;
}

/**
 * Get player's horizontal facing direction
 * @param {Player} player
 * @returns {{dx: number, dz: number, rotation: number}} Unit direction and rotation (0=S, 90=W, 180=N, 270=E)
 */
export function getPlayerFacing(player) {
    const rotation = player.getRotation();
    const yaw = rotation.y; // -180 to 180, 0 = south (+Z)

    // Normalize to 0-360
    const normalizedYaw = ((yaw % 360) + 360) % 360;

    // Determine cardinal direction
    if (normalizedYaw >= 315 || normalizedYaw < 45) {
        return { dx: 0, dz: 1, rotation: 0 };     // South (+Z)
    } else if (normalizedYaw >= 45 && normalizedYaw < 135) {
        return { dx: -1, dz: 0, rotation: 90 };   // West (-X)
    } else if (normalizedYaw >= 135 && normalizedYaw < 225) {
        return { dx: 0, dz: -1, rotation: 180 };  // North (-Z)
    } else {
        return { dx: 1, dz: 0, rotation: 270 };   // East (+X)
    }
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
 * Get rotated size after rotation
 * @param {number[]} size - [width, height, length]
 * @param {number} rotation - 0, 90, 180, or 270 degrees
 * @returns {number[]} - [newWidth, height, newLength]
 */
export function getRotatedSize(size, rotation) {
    const [w, h, l] = size;
    if (rotation === 90 || rotation === 270) {
        return [l, h, w];
    }
    return [w, h, l];
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

    // Check for stored marker
    const marker = getMarker();
    let buildX, buildY, buildZ, rotation, dimension;

    if (marker) {
        buildX = marker.x;
        buildY = marker.y;
        buildZ = marker.z;
        rotation = marker.rotation;
        dimension = marker.dimension;
        player.sendMessage(`§aBuilding ${width}x${height}x${length} maze at marker...`);
        // Clear the marker after use
        clearMarker();
    } else {
        // Fall back to player position and facing
        const pos = player.location;
        buildX = Math.floor(pos.x);
        buildY = Math.floor(pos.y);
        buildZ = Math.floor(pos.z);
        rotation = getPlayerFacing(player).rotation;
        dimension = player.dimension;
        player.sendMessage(`§aGenerating ${width}x${height}x${length} maze at player...`);
    }

    // Generate the maze
    const { grid, size } = generateMazeGrid(width, height, length);

    // Convert to blocks
    let blocks = gridToBlocks(grid, blockType);

    // Rotate maze based on marker/player rotation
    blocks = rotateBlocks(blocks, size, rotation);
    const rotatedSize = getRotatedSize(size, rotation);
    const [rw, rh, rl] = rotatedSize;

    const totalBlocks = blocks.length;
    player.sendMessage(`§7Placing ${totalBlocks} blocks...`);

    // Build at the target location
    // Origin is [0, 0, 0] - maze builds with corner at marker/player position
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
