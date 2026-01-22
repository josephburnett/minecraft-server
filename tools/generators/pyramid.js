#!/usr/bin/env node
/**
 * Pyramid Generator
 * Generates step pyramids with player at base center
 *
 * Usage: node pyramid.js [base-size] [block]
 * Example: node pyramid.js 15 minecraft:sandstone
 *
 * Outputs a scriptevent command to generate the pyramid
 */

const { createBitfieldStructure, toChunks, createGrid } = require('../lib/structure.js');

// Parse arguments
const args = process.argv.slice(2);
const baseSize = parseInt(args[0]) || 15;
const block = args[1] || "minecraft:sandstone";

// Ensure odd base size for proper centering
const base = baseSize % 2 === 0 ? baseSize + 1 : baseSize;

// Calculate height (each level shrinks by 2, so height = (base + 1) / 2)
const height = Math.floor((base + 1) / 2);

// Initialize grid
const grid = createGrid(base, height, base, false);

// Generate pyramid layer by layer
for (let y = 0; y < height; y++) {
    // Each layer is smaller by 2 (1 on each side)
    const layerSize = base - (y * 2);
    const offset = y; // Offset from edge

    for (let x = offset; x < base - offset; x++) {
        for (let z = offset; z < base - offset; z++) {
            grid[x][y][z] = true;
        }
    }
}

// Origin: player at center of base (ground level)
const center = Math.floor(base / 2);
const origin = [center, 0, center];

// Create and output the structure
const structure = createBitfieldStructure(
    [base, height, base],
    origin,
    block,
    grid
);

toChunks(structure).forEach(chunk => console.log(chunk));

// Print info to stderr
console.error(`Generated pyramid with base ${base}x${base} and height ${height}`);
console.error(`Block: ${block}`);
console.error(`Player spawns at base center`);
