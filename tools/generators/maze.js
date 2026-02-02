#!/usr/bin/env node
/**
 * 3D Maze Generator
 * Uses recursive backtracking algorithm
 *
 * Usage: node maze.js [width] [height] [length] [block]
 * Example: node maze.js 15 7 15 minecraft:stone
 *
 * Outputs a scriptevent command to generate the maze
 */

import { createBitfieldStructure, toChunks, createGrid } from '../lib/structure.js';

// Parse arguments
const args = process.argv.slice(2);
const width = parseInt(args[0]) || 15;   // X dimension (odd numbers work best)
const height = parseInt(args[1]) || 7;   // Y dimension
const length = parseInt(args[2]) || 15;  // Z dimension (odd numbers work best)
const block = args[3] || "minecraft:stone_bricks";

// Ensure odd dimensions for proper maze walls
const mazeW = width % 2 === 0 ? width + 1 : width;
const mazeH = height % 2 === 0 ? height + 1 : height;
const mazeL = length % 2 === 0 ? length + 1 : length;

// Cell dimensions (maze cells are 2x2xHeight with walls between)
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

// Shuffle array in place
function shuffle(array) {
    for (let i = array.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [array[i], array[j]] = [array[j], array[i]];
    }
    return array;
}

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

// Recursive backtracking maze generation
function generateMaze(cx, cz) {
    visited[cx][cz] = true;
    const [gx, gz] = cellToGrid(cx, cz);
    carveCell(gx, gz);

    // Randomize direction order
    const dirs = shuffle([...directions]);

    for (const [dx, dz] of dirs) {
        const nx = cx + dx;
        const nz = cz + dz;

        // Check bounds
        if (nx < 0 || nx >= cellsX || nz < 0 || nz >= cellsZ) continue;

        // Check if already visited
        if (visited[nx][nz]) continue;

        // Carve passage and recurse
        carvePassage(cx, cz, nx, nz);
        generateMaze(nx, nz);
    }
}

// Add vertical passages (optional, for multi-level mazes)
function addVerticalPassages() {
    if (mazeH <= 3) return; // Need at least 4 height for vertical passages

    // Add some random vertical openings
    const numOpenings = Math.floor(cellsX * cellsZ * 0.1); // 10% of cells

    for (let i = 0; i < numOpenings; i++) {
        const cx = Math.floor(Math.random() * cellsX);
        const cz = Math.floor(Math.random() * cellsZ);
        const [gx, gz] = cellToGrid(cx, cz);

        // Carve a hole in the floor (but not through bottom)
        if (grid[gx][1][gz] === false) { // If this cell is a passage
            // Don't carve through floor to avoid falling into void
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

// Origin: player starts at entrance
// Entrance is at grid position [1, 1, 0], so origin places player there
const origin = [1, 1, 0];

// Create and output the structure
const structure = createBitfieldStructure(
    [mazeW, mazeH, mazeL],
    origin,
    block,
    grid
);

toChunks(structure).forEach(chunk => console.log(chunk));

// Print info to stderr so it doesn't interfere with the command output
console.error(`Generated ${mazeW}x${mazeH}x${mazeL} maze with ${block}`);
console.error(`Maze cells: ${cellsX}x${cellsZ}`);
console.error(`Player spawns at entrance`);
