import { describe, it, expect } from 'vitest';
import {
    createGrid,
    gridToBlocks,
    generateSphereGrid,
    generateCubeGrid
} from '../behavior_pack/scripts/shapes.js';

describe('createGrid', () => {
    it('creates grid with correct dimensions', () => {
        const grid = createGrid(3, 4, 5, false);
        expect(grid.length).toBe(3);
        expect(grid[0].length).toBe(4);
        expect(grid[0][0].length).toBe(5);
    });

    it('initializes with specified value', () => {
        const grid = createGrid(2, 2, 2, true);
        expect(grid[0][0][0]).toBe(true);
        expect(grid[1][1][1]).toBe(true);
    });

    it('defaults to false', () => {
        const grid = createGrid(2, 2, 2);
        expect(grid[0][0][0]).toBe(false);
    });
});

describe('gridToBlocks', () => {
    it('converts grid to block array', () => {
        const grid = createGrid(2, 2, 2, false);
        grid[0][0][0] = true;
        grid[1][1][1] = true;

        const blocks = gridToBlocks(grid, 'minecraft:stone');

        expect(blocks.length).toBe(2);
        expect(blocks).toContainEqual([0, 0, 0, 'minecraft:stone']);
        expect(blocks).toContainEqual([1, 1, 1, 'minecraft:stone']);
    });

    it('returns empty array for empty grid', () => {
        const grid = createGrid(2, 2, 2, false);
        const blocks = gridToBlocks(grid, 'minecraft:stone');
        expect(blocks.length).toBe(0);
    });
});

describe('generateSphereGrid', () => {
    it('generates solid sphere with correct size', () => {
        const { grid, size, center } = generateSphereGrid(3, false);

        expect(size).toBe(7); // 2 * 3 + 1
        expect(center).toBe(3);
        expect(grid.length).toBe(7);
    });

    it('solid sphere has center block', () => {
        const { grid, center } = generateSphereGrid(3, false);
        expect(grid[center][center][center]).toBe(true);
    });

    it('solid sphere has block at radius distance', () => {
        const { grid, center } = generateSphereGrid(3, false);
        // Block at exactly radius distance on X axis
        expect(grid[center + 3][center][center]).toBe(true);
    });

    it('solid sphere does not have block beyond radius', () => {
        const { grid, center } = generateSphereGrid(3, false);
        // This would be at the corner, distance > radius
        expect(grid[0][0][0]).toBe(false);
    });

    it('hollow sphere has center empty', () => {
        const { grid, center } = generateSphereGrid(5, true);
        expect(grid[center][center][center]).toBe(false);
    });

    it('hollow sphere has surface blocks', () => {
        const { grid, center } = generateSphereGrid(5, true);
        // Surface at radius distance
        expect(grid[center + 5][center][center]).toBe(true);
    });

    it('radius 1 solid sphere has correct block count', () => {
        const { grid } = generateSphereGrid(1, false);
        const blocks = gridToBlocks(grid, 'stone');
        // Radius 1 sphere: center + 6 adjacent = 7 blocks
        expect(blocks.length).toBe(7);
    });
});

describe('generateCubeGrid', () => {
    it('generates cube with correct size', () => {
        const { grid, size, center } = generateCubeGrid(5, false);

        expect(size).toBe(5);
        expect(center).toBe(2); // floor(5/2)
        expect(grid.length).toBe(5);
    });

    it('solid cube has all blocks', () => {
        const { grid, size } = generateCubeGrid(3, false);
        const blocks = gridToBlocks(grid, 'stone');
        expect(blocks.length).toBe(27); // 3^3
    });

    it('hollow cube has only surface blocks', () => {
        const { grid, size } = generateCubeGrid(3, true);
        const blocks = gridToBlocks(grid, 'stone');
        // 3x3x3 cube: 27 total - 1 interior = 26 surface
        expect(blocks.length).toBe(26);
    });

    it('hollow cube has empty center', () => {
        const { grid, center } = generateCubeGrid(5, true);
        // Center of 5x5x5 cube is [2,2,2]
        expect(grid[center][center][center]).toBe(false);
    });

    it('hollow cube has corner blocks', () => {
        const { grid, size } = generateCubeGrid(5, true);
        expect(grid[0][0][0]).toBe(true);
        expect(grid[size-1][size-1][size-1]).toBe(true);
    });

    it('larger hollow cube has more empty interior', () => {
        const { grid } = generateCubeGrid(5, true);
        const blocks = gridToBlocks(grid, 'stone');
        // 5x5x5 = 125, interior is 3x3x3 = 27, surface = 98
        expect(blocks.length).toBe(98);
    });
});

describe('sphere vs cube block counts', () => {
    it('solid cube has more blocks than inscribed sphere', () => {
        // For same "size", cube has more blocks
        const sphere = generateSphereGrid(5, false);
        const cube = generateCubeGrid(11, false); // 11 = 2*5+1 to match sphere grid size

        const sphereBlocks = gridToBlocks(sphere.grid, 'stone').length;
        const cubeBlocks = gridToBlocks(cube.grid, 'stone').length;

        expect(cubeBlocks).toBeGreaterThan(sphereBlocks);
    });
});
