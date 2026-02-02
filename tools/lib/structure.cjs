/**
 * Shared structure encoding functions for Minecraft structure generators
 */

/**
 * Encode a 3D boolean grid as a base64-encoded bitfield
 * @param {boolean[][][]} grid - 3D array [x][y][z] where true = block, false = air
 * @returns {string} base64-encoded bitfield
 */
function encodeBitfield(grid) {
    const width = grid.length;
    const height = grid[0]?.length || 0;
    const length = grid[0]?.[0]?.length || 0;

    const bits = [];
    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                bits.push(grid[x][y][z] ? 1 : 0);
            }
        }
    }

    // Pack bits into bytes
    const bytes = [];
    for (let i = 0; i < bits.length; i += 8) {
        let byte = 0;
        for (let j = 0; j < 8 && i + j < bits.length; j++) {
            byte |= bits[i + j] << (7 - j);
        }
        bytes.push(byte);
    }

    return Buffer.from(bytes).toString('base64');
}

/**
 * Encode a 3D grid with palette indices as base64
 * @param {number[][][]} grid - 3D array [x][y][z] of palette indices
 * @param {string[]} palette - array of block type IDs
 * @returns {string} base64-encoded indices
 */
function encodePalette(grid, palette) {
    const width = grid.length;
    const height = grid[0]?.length || 0;
    const length = grid[0]?.[0]?.length || 0;

    const bytes = [];
    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                bytes.push(grid[x][y][z]);
            }
        }
    }

    return Buffer.from(bytes).toString('base64');
}

/**
 * Create a bitfield structure object
 * @param {number[]} size - [width, height, length]
 * @param {number[]} origin - [x, y, z] player position relative to structure [0,0,0]
 * @param {string} block - block type for 1-bits
 * @param {boolean[][][]} grid - 3D boolean grid
 * @returns {object} structure object
 */
function createBitfieldStructure(size, origin, block, grid) {
    return {
        type: "bitfield",
        size: size,
        origin: origin,
        block: block,
        data: encodeBitfield(grid)
    };
}

/**
 * Create a palette structure object
 * @param {number[]} size - [width, height, length]
 * @param {number[]} origin - [x, y, z] player position relative to structure [0,0,0]
 * @param {string[]} palette - array of block type IDs
 * @param {number[][][]} grid - 3D array of palette indices
 * @returns {object} structure object
 */
function createPaletteStructure(size, origin, palette, grid) {
    return {
        type: "palette",
        size: size,
        origin: origin,
        palette: palette,
        data: encodePalette(grid, palette)
    };
}

/**
 * Create a sparse structure object
 * @param {number[]} origin - [x, y, z] player position relative to structure [0,0,0]
 * @param {Array<[number, number, number, string]>} blocks - array of [x, y, z, blockType]
 * @returns {object} structure object
 */
function createSparseStructure(origin, blocks) {
    return {
        type: "sparse",
        origin: origin,
        blocks: blocks
    };
}

/**
 * Convert a structure object to chunk data lines for upload
 * Always chunks the data regardless of size for consistent handling
 * @param {object} structure - structure object
 * @returns {string[]} array of chunk data lines (sessionId:index:total:data)
 */
function toChunks(structure) {
    const json = JSON.stringify(structure);
    const base64 = Buffer.from(json).toString('base64');
    // Bedrock Realms enforce a 512-character limit on commands.
    // Full command: /scriptevent burnodd:chunk {sessionId}:{i}:{total}:{data}
    // Overhead is ~45 chars, leaving ~467 for data. Use 400 for safety.
    const maxChunkSize = 400;

    const sessionId = Math.random().toString(36).substring(2, 8);
    const chunks = [];
    for (let i = 0; i < base64.length; i += maxChunkSize) {
        chunks.push(base64.substring(i, i + maxChunkSize));
    }

    // Return line-delimited chunk data (no scriptevent prefix)
    return chunks.map((data, i) => `${sessionId}:${i}:${chunks.length}:${data}`);
}

/**
 * Create a 3D boolean grid initialized to a value
 * @param {number} width
 * @param {number} height
 * @param {number} length
 * @param {boolean} value - initial value (default false)
 * @returns {boolean[][][]}
 */
function createGrid(width, height, length, value = false) {
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
 * Create a 3D numeric grid initialized to a value
 * @param {number} width
 * @param {number} height
 * @param {number} length
 * @param {number} value - initial value (default 0)
 * @returns {number[][][]}
 */
function createNumericGrid(width, height, length, value = 0) {
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
 * Convert a structure to block placement lines (CSV format: x,y,z,block_name).
 * Used for direct block placement via InventoryTransaction packets.
 * @param {object} structure - structure object (bitfield, palette, or sparse)
 * @returns {string[]} array of "x,y,z,block_name" lines
 */
function toBlocks(structure) {
    const lines = [];

    if (structure.type === "sparse") {
        for (const [x, y, z, blockName] of structure.blocks) {
            lines.push(`${x},${y},${z},${blockName}`);
        }
    } else if (structure.type === "bitfield") {
        const [width, height, length] = structure.size;
        // Decode the bitfield back to get block positions
        const bytes = Buffer.from(structure.data, 'base64');
        let bitIndex = 0;
        for (let x = 0; x < width; x++) {
            for (let y = 0; y < height; y++) {
                for (let z = 0; z < length; z++) {
                    const byteIdx = Math.floor(bitIndex / 8);
                    const bitIdx = 7 - (bitIndex % 8);
                    if (byteIdx < bytes.length && (bytes[byteIdx] >> bitIdx) & 1) {
                        lines.push(`${x},${y},${z},${structure.block}`);
                    }
                    bitIndex++;
                }
            }
        }
    } else if (structure.type === "palette") {
        const [width, height, length] = structure.size;
        const bytes = Buffer.from(structure.data, 'base64');
        let idx = 0;
        for (let x = 0; x < width; x++) {
            for (let y = 0; y < height; y++) {
                for (let z = 0; z < length; z++) {
                    const paletteIdx = bytes[idx];
                    if (paletteIdx > 0 && paletteIdx < structure.palette.length) {
                        lines.push(`${x},${y},${z},${structure.palette[paletteIdx]}`);
                    }
                    idx++;
                }
            }
        }
    }

    return lines;
}

/**
 * Convert a 3D boolean grid directly to block placement lines.
 * More efficient than going through the bitfield encoding.
 * @param {boolean[][][]} grid - 3D array [x][y][z]
 * @param {string} block - block type name
 * @returns {string[]} array of "x,y,z,block_name" lines
 */
function gridToBlocks(grid, block) {
    const lines = [];
    const width = grid.length;
    const height = grid[0]?.length || 0;
    const length = grid[0]?.[0]?.length || 0;

    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                if (grid[x][y][z]) {
                    lines.push(`${x},${y},${z},${block}`);
                }
            }
        }
    }
    return lines;
}

module.exports = {
    encodeBitfield,
    encodePalette,
    createBitfieldStructure,
    createPaletteStructure,
    createSparseStructure,
    toChunks,
    toBlocks,
    gridToBlocks,
    createGrid,
    createNumericGrid
};
