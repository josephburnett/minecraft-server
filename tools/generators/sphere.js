#!/usr/bin/env node
/**
 * Sphere Generator
 * Generates solid or hollow spheres centered on player
 *
 * Usage: node sphere.js [radius] [block] [hollow]
 * Example: node sphere.js 5 minecraft:glass true
 *
 * Outputs a scriptevent command to generate the sphere
 */

import { createBitfieldStructure, toChunks, createGrid } from '../lib/structure.js';

// Parse arguments
const args = process.argv.slice(2);
const radius = parseInt(args[0]) || 5;
const block = args[1] || "minecraft:glass";
const hollow = args[2] === "true" || args[2] === "hollow";

// Calculate dimensions (diameter = 2 * radius + 1 for center block)
const size = radius * 2 + 1;

// Initialize grid
const grid = createGrid(size, size, size, false);

// Center point in the grid
const center = radius;

// Generate sphere
for (let x = 0; x < size; x++) {
    for (let y = 0; y < size; y++) {
        for (let z = 0; z < size; z++) {
            // Distance from center
            const dx = x - center;
            const dy = y - center;
            const dz = z - center;
            const distSq = dx * dx + dy * dy + dz * dz;

            if (hollow) {
                // Hollow sphere: only blocks at the surface
                // Inner radius is radius - 1, outer radius is radius
                const innerRadiusSq = (radius - 1) * (radius - 1);
                const outerRadiusSq = radius * radius;
                if (distSq <= outerRadiusSq && distSq > innerRadiusSq) {
                    grid[x][y][z] = true;
                }
            } else {
                // Solid sphere: all blocks within radius
                if (distSq <= radius * radius) {
                    grid[x][y][z] = true;
                }
            }
        }
    }
}

// Origin: player at center of sphere
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
console.error(`Generated ${hollow ? "hollow" : "solid"} sphere with radius ${radius}`);
console.error(`Size: ${size}x${size}x${size}`);
console.error(`Block: ${block}`);
console.error(`Player spawns at center`);
