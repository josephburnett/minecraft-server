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
 * Convert a structure object to a scriptevent command
 * @param {object} structure - structure object
 * @returns {string} complete scriptevent command
 */
function toCommand(structure) {
    const json = JSON.stringify(structure);
    const base64 = Buffer.from(json).toString('base64');
    return `scriptevent family:build ${base64}`;
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

module.exports = {
    encodeBitfield,
    encodePalette,
    createBitfieldStructure,
    createPaletteStructure,
    createSparseStructure,
    toCommand,
    createGrid,
    createNumericGrid
};
