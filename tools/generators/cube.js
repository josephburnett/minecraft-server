#!/usr/bin/env node
/**
 * Cube Generator
 * Generates solid or hollow cubes centered on player
 *
 * Usage: node cube.js [size] [block] [hollow]
 * Example: node cube.js 10 minecraft:stone true
 *
 * Outputs a scriptevent command to generate the cube
 */

const { createBitfieldStructure, toChunks, createGrid } = require('../lib/structure.js');

// Parse arguments
const args = process.argv.slice(2);
const size = parseInt(args[0]) || 10;
const block = args[1] || "minecraft:stone";
const hollow = args[2] === "true" || args[2] === "hollow";

// Initialize grid
const grid = createGrid(size, size, size, false);

// Generate cube
for (let x = 0; x < size; x++) {
    for (let y = 0; y < size; y++) {
        for (let z = 0; z < size; z++) {
            if (hollow) {
                // Hollow cube: only blocks on the surface
                const onSurface = (
                    x === 0 || x === size - 1 ||
                    y === 0 || y === size - 1 ||
                    z === 0 || z === size - 1
                );
                if (onSurface) {
                    grid[x][y][z] = true;
                }
            } else {
                // Solid cube: all blocks
                grid[x][y][z] = true;
            }
        }
    }
}

// Origin: player at center of cube
// For even sizes, center is size/2, for odd it's (size-1)/2
const center = Math.floor(size / 2);
const origin = [center, center, center];

// Create and output the structure
const structure = createBitfieldStructure(
    [size, size, size],
    origin,
    block,
    grid
);

toChunks(structure).forEach(chunk => console.log(chunk));

// Print info to stderr
console.error(`Generated ${hollow ? "hollow" : "solid"} cube of size ${size}`);
console.error(`Block: ${block}`);
console.error(`Player spawns at center`);
