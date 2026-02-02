#!/usr/bin/env node
/**
 * Test Pattern Generator
 * Generates simple test patterns for debugging structure import
 *
 * Usage: node test.js [pattern] [size]
 * Patterns: checkerboard, stripes, frame, cross, corner
 * Example: node test.js checkerboard 10
 *
 * Outputs a scriptevent command to generate the pattern
 */

const { createSparseStructure, toChunks } = require('../lib/structure.js');

// Parse arguments
const args = process.argv.slice(2);
const pattern = args[0] || "frame";
const size = parseInt(args[1]) || 10;

// Generate blocks based on pattern
let blocks = [];
let origin = [0, 0, 0];
let description = "";

switch (pattern) {
    case "checkerboard":
        // 2D checkerboard pattern on the ground
        for (let x = 0; x < size; x++) {
            for (let z = 0; z < size; z++) {
                if ((x + z) % 2 === 0) {
                    blocks.push([x, 0, z, "minecraft:white_concrete"]);
                } else {
                    blocks.push([x, 0, z, "minecraft:black_concrete"]);
                }
            }
        }
        origin = [Math.floor(size / 2), 0, Math.floor(size / 2)];
        description = `${size}x${size} checkerboard`;
        break;

    case "stripes":
        // Vertical stripes
        const colors = [
            "minecraft:red_concrete",
            "minecraft:orange_concrete",
            "minecraft:yellow_concrete",
            "minecraft:lime_concrete",
            "minecraft:blue_concrete",
            "minecraft:purple_concrete"
        ];
        for (let x = 0; x < size; x++) {
            for (let z = 0; z < size; z++) {
                const colorIdx = x % colors.length;
                blocks.push([x, 0, z, colors[colorIdx]]);
            }
        }
        origin = [Math.floor(size / 2), 0, Math.floor(size / 2)];
        description = `${size}x${size} rainbow stripes`;
        break;

    case "frame":
        // 3D wireframe cube
        for (let x = 0; x < size; x++) {
            for (let y = 0; y < size; y++) {
                for (let z = 0; z < size; z++) {
                    // Only edges of the cube
                    const onXEdge = (x === 0 || x === size - 1);
                    const onYEdge = (y === 0 || y === size - 1);
                    const onZEdge = (z === 0 || z === size - 1);

                    // An edge requires at least 2 of 3 coordinates to be at extremes
                    const edgeCount = (onXEdge ? 1 : 0) + (onYEdge ? 1 : 0) + (onZEdge ? 1 : 0);
                    if (edgeCount >= 2) {
                        blocks.push([x, y, z, "minecraft:iron_block"]);
                    }
                }
            }
        }
        origin = [Math.floor(size / 2), Math.floor(size / 2), Math.floor(size / 2)];
        description = `${size}x${size}x${size} wireframe`;
        break;

    case "cross":
        // 3D cross/plus shape
        const center = Math.floor(size / 2);
        for (let i = 0; i < size; i++) {
            // X axis
            blocks.push([i, center, center, "minecraft:red_concrete"]);
            // Y axis
            if (i !== center) blocks.push([center, i, center, "minecraft:lime_concrete"]);
            // Z axis
            if (i !== center) blocks.push([center, center, i, "minecraft:blue_concrete"]);
        }
        origin = [center, center, center];
        description = `${size}-length 3D cross (RGB axes)`;
        break;

    case "corner":
        // L-shaped corner marker to verify orientation
        // X axis = red, Y axis = green, Z axis = blue
        for (let i = 0; i < 5; i++) {
            blocks.push([i, 0, 0, "minecraft:red_concrete"]);
            blocks.push([0, i, 0, "minecraft:lime_concrete"]);
            blocks.push([0, 0, i, "minecraft:blue_concrete"]);
        }
        // Add white block at origin for reference
        blocks.push([0, 0, 0, "minecraft:white_concrete"]);
        origin = [0, 0, 0];
        description = "Corner marker (X=red, Y=green, Z=blue)";
        break;

    case "small":
        // Minimal test: single block
        blocks.push([0, 0, 0, "minecraft:diamond_block"]);
        origin = [0, 0, 0];
        description = "Single diamond block at player position";
        break;

    case "line":
        // Simple line of blocks in front of player
        for (let z = 1; z <= 10; z++) {
            blocks.push([0, 0, z, "minecraft:gold_block"]);
        }
        origin = [0, 0, 0];
        description = "Line of 10 gold blocks in front of player";
        break;

    default:
        console.error(`Unknown pattern: ${pattern}`);
        console.error("Available patterns: checkerboard, stripes, frame, cross, corner, small, line");
        process.exit(1);
}

// Create and output the structure
const structure = createSparseStructure(origin, blocks);

toChunks(structure).forEach(chunk => console.log(chunk));

// Print info to stderr
console.error(`Generated test pattern: ${description}`);
console.error(`Total blocks: ${blocks.length}`);
