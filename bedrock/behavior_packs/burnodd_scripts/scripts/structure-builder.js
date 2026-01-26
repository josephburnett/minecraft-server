import { system } from "@minecraft/server";

/**
 * Base64 decode (Bedrock JS doesn't have atob)
 * @param {string} str - Base64 encoded string
 * @returns {number[]} Array of bytes
 */
export function base64Decode(str) {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let result = [];
    let buffer = 0;
    let bits = 0;

    for (let i = 0; i < str.length; i++) {
        const c = str[i];
        if (c === "=") break;
        const idx = chars.indexOf(c);
        if (idx === -1) continue;

        buffer = (buffer << 6) | idx;
        bits += 6;

        while (bits >= 8) {
            bits -= 8;
            result.push((buffer >> bits) & 0xff);
        }
    }

    return result;
}

/**
 * Decode bitfield format
 * @param {string} data - Base64 encoded bitfield
 * @param {number[]} size - [width, height, length]
 * @returns {Array<[number, number, number]>} Array of [x, y, z] positions
 */
export function decodeBitfield(data, size) {
    const bytes = base64Decode(data);
    const [width, height, length] = size;
    const blocks = [];

    let bitIndex = 0;
    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                const byteIndex = Math.floor(bitIndex / 8);
                const bitOffset = 7 - (bitIndex % 8);
                const bit = (bytes[byteIndex] >> bitOffset) & 1;

                if (bit === 1) {
                    blocks.push([x, y, z]);
                }
                bitIndex++;
            }
        }
    }

    return blocks;
}

/**
 * Decode palette format
 * @param {string} data - Base64 encoded palette indices
 * @param {number[]} size - [width, height, length]
 * @param {string[]} palette - Array of block type IDs
 * @returns {Array<[number, number, number, string]>} Array of [x, y, z, blockType]
 */
export function decodePalette(data, size, palette) {
    const bytes = base64Decode(data);
    const [width, height, length] = size;
    const blocks = [];

    let index = 0;
    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                const paletteIndex = bytes[index];
                const blockType = palette[paletteIndex];

                if (blockType && blockType !== "minecraft:air") {
                    blocks.push([x, y, z, blockType]);
                }
                index++;
            }
        }
    }

    return blocks;
}

/**
 * Build a structure at player location
 * @param {Player} player
 * @param {object} structure - Structure definition
 */
export function buildStructure(player, structure) {
    const dimension = player.dimension;
    const playerPos = player.location;
    const px = Math.floor(playerPos.x);
    const py = Math.floor(playerPos.y);
    const pz = Math.floor(playerPos.z);

    const origin = structure.origin || [0, 0, 0];
    let blocks = [];

    // Decode based on structure type
    if (structure.type === "bitfield") {
        const positions = decodeBitfield(structure.data, structure.size);
        const block = structure.block || "minecraft:stone";
        blocks = positions.map(([x, y, z]) => [x, y, z, block]);
    } else if (structure.type === "palette") {
        blocks = decodePalette(structure.data, structure.size, structure.palette);
    } else if (structure.type === "sparse") {
        blocks = structure.blocks || [];
    } else {
        player.sendMessage(`§cUnknown structure type: ${structure.type}`);
        return;
    }

    if (blocks.length === 0) {
        player.sendMessage(`§cNo blocks to place!`);
        return;
    }

    player.sendMessage(`§aBuilding structure: §f${blocks.length} blocks...`);

    // Async chunked building to avoid watchdog
    const blocksPerTick = 1000;
    let index = 0;
    let placed = 0;

    const intervalId = system.runInterval(() => {
        const endIndex = Math.min(index + blocksPerTick, blocks.length);

        for (; index < endIndex; index++) {
            const [sx, sy, sz, blockType] = blocks[index];

            const wx = px + sx - origin[0];
            const wy = py + sy - origin[1];
            const wz = pz + sz - origin[2];

            try {
                const block = dimension.getBlock({ x: wx, y: wy, z: wz });
                if (block) {
                    block.setType(blockType);
                    placed++;
                }
            } catch (e) {
                // Block might be outside loaded chunks
            }
        }

        if (index >= blocks.length) {
            system.clearRun(intervalId);
            player.sendMessage(`§a§lBuild complete! §r§7Placed ${placed} blocks`);
        }
    }, 1);
}

/**
 * Build blocks directly at player location (no encoding)
 * @param {Player} player
 * @param {Array<[number, number, number, string]>} blocks - Array of [x, y, z, blockType]
 * @param {number[]} origin - [x, y, z] offset from player position
 * @param {function} [onComplete] - Optional callback when build completes
 */
export function buildBlocks(player, blocks, origin = [0, 0, 0], onComplete = null) {
    const dimension = player.dimension;
    const playerPos = player.location;
    const px = Math.floor(playerPos.x);
    const py = Math.floor(playerPos.y);
    const pz = Math.floor(playerPos.z);

    if (blocks.length === 0) {
        player.sendMessage(`§cNo blocks to place!`);
        return;
    }

    player.sendMessage(`§aBuilding: §f${blocks.length} blocks...`);

    const blocksPerTick = 1000;
    let index = 0;
    let placed = 0;

    const intervalId = system.runInterval(() => {
        const endIndex = Math.min(index + blocksPerTick, blocks.length);

        for (; index < endIndex; index++) {
            const [sx, sy, sz, blockType] = blocks[index];

            const wx = px + sx - origin[0];
            const wy = py + sy - origin[1];
            const wz = pz + sz - origin[2];

            try {
                const block = dimension.getBlock({ x: wx, y: wy, z: wz });
                if (block) {
                    block.setType(blockType);
                    placed++;
                }
            } catch (e) {
                // Block might be outside loaded chunks
            }
        }

        if (index >= blocks.length) {
            system.clearRun(intervalId);
            player.sendMessage(`§a§lBuild complete! §r§7Placed ${placed} blocks`);
            if (onComplete) onComplete(placed);
        }
    }, 1);
}

/**
 * Build blocks at a specific world location
 * @param {Dimension} dimension
 * @param {number} baseX - Base X coordinate
 * @param {number} baseY - Base Y coordinate
 * @param {number} baseZ - Base Z coordinate
 * @param {Array<[number, number, number, string]>} blocks - Array of [x, y, z, blockType]
 * @param {function} [onComplete] - Optional callback when build completes
 */
export function buildBlocksAt(dimension, baseX, baseY, baseZ, blocks, onComplete = null) {
    if (blocks.length === 0) {
        return;
    }

    const blocksPerTick = 1000;
    let index = 0;
    let placed = 0;

    const intervalId = system.runInterval(() => {
        const endIndex = Math.min(index + blocksPerTick, blocks.length);

        for (; index < endIndex; index++) {
            const [sx, sy, sz, blockType] = blocks[index];

            const wx = baseX + sx;
            const wy = baseY + sy;
            const wz = baseZ + sz;

            try {
                const block = dimension.getBlock({ x: wx, y: wy, z: wz });
                if (block) {
                    block.setType(blockType);
                    placed++;
                }
            } catch (e) {
                // Block might be outside loaded chunks
            }
        }

        if (index >= blocks.length) {
            system.clearRun(intervalId);
            if (onComplete) onComplete(placed);
        }
    }, 1);
}
