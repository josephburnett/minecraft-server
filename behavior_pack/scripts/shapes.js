/**
 * Shared shape generation utilities
 */

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
 * Convert a boolean grid to block array
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
 * Generate a sphere grid
 * @param {number} radius
 * @param {boolean} hollow - If true, only surface blocks
 * @returns {{grid: boolean[][][], size: number, center: number}}
 */
export function generateSphereGrid(radius, hollow = false) {
    const size = radius * 2 + 1;
    const grid = createGrid(size, size, size, false);
    const center = radius;

    for (let x = 0; x < size; x++) {
        for (let y = 0; y < size; y++) {
            for (let z = 0; z < size; z++) {
                const dx = x - center;
                const dy = y - center;
                const dz = z - center;
                const distSq = dx * dx + dy * dy + dz * dz;

                if (hollow) {
                    const innerRadiusSq = (radius - 1) * (radius - 1);
                    const outerRadiusSq = radius * radius;
                    if (distSq <= outerRadiusSq && distSq > innerRadiusSq) {
                        grid[x][y][z] = true;
                    }
                } else {
                    if (distSq <= radius * radius) {
                        grid[x][y][z] = true;
                    }
                }
            }
        }
    }

    return { grid, size, center };
}

/**
 * Generate a cube grid
 * @param {number} size
 * @param {boolean} hollow - If true, only surface blocks
 * @returns {{grid: boolean[][][], size: number, center: number}}
 */
export function generateCubeGrid(size, hollow = false) {
    const grid = createGrid(size, size, size, false);
    const center = Math.floor(size / 2);

    for (let x = 0; x < size; x++) {
        for (let y = 0; y < size; y++) {
            for (let z = 0; z < size; z++) {
                if (hollow) {
                    const onSurface = (
                        x === 0 || x === size - 1 ||
                        y === 0 || y === size - 1 ||
                        z === 0 || z === size - 1
                    );
                    if (onSurface) {
                        grid[x][y][z] = true;
                    }
                } else {
                    grid[x][y][z] = true;
                }
            }
        }
    }

    return { grid, size, center };
}

/**
 * Get player's horizontal facing as rotation (0, 90, 180, 270)
 * @param {Player} player
 * @returns {number}
 */
export function getPlayerRotation(player) {
    const rotation = player.getRotation();
    const yaw = rotation.y;
    const normalizedYaw = ((yaw % 360) + 360) % 360;

    if (normalizedYaw >= 315 || normalizedYaw < 45) {
        return 0;     // South (+Z)
    } else if (normalizedYaw >= 45 && normalizedYaw < 135) {
        return 270;   // West (-X)
    } else if (normalizedYaw >= 135 && normalizedYaw < 225) {
        return 180;   // North (-Z)
    } else {
        return 90;    // East (+X)
    }
}
