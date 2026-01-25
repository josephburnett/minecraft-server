import { describe, it, expect } from 'vitest';
import {
    createGrid,
    shuffle,
    generateMazeGrid,
    gridToBlocks,
    rotateBlocks,
    getRotatedSize,
    parseMazeOptions
} from '../bedrock/behavior_packs/burnodd_scripts/scripts/maze.js';

describe('createGrid', () => {
    it('creates a 3D grid with correct dimensions', () => {
        const grid = createGrid(3, 4, 5, false);

        expect(grid.length).toBe(3);
        expect(grid[0].length).toBe(4);
        expect(grid[0][0].length).toBe(5);
    });

    it('initializes all values to the specified value', () => {
        const grid = createGrid(2, 2, 2, true);

        for (let x = 0; x < 2; x++) {
            for (let y = 0; y < 2; y++) {
                for (let z = 0; z < 2; z++) {
                    expect(grid[x][y][z]).toBe(true);
                }
            }
        }
    });

    it('defaults to false when no value specified', () => {
        const grid = createGrid(2, 2, 2);
        expect(grid[0][0][0]).toBe(false);
    });
});

describe('shuffle', () => {
    it('returns an array of the same length', () => {
        const arr = [1, 2, 3, 4, 5];
        const shuffled = shuffle([...arr]);
        expect(shuffled.length).toBe(arr.length);
    });

    it('contains all original elements', () => {
        const arr = [1, 2, 3, 4, 5];
        const shuffled = shuffle([...arr]);
        expect(shuffled.sort()).toEqual(arr.sort());
    });

    it('modifies the array in place', () => {
        const arr = [1, 2, 3, 4, 5];
        const result = shuffle(arr);
        expect(result).toBe(arr);
    });
});

describe('generateMazeGrid', () => {
    it('returns a grid with correct structure', () => {
        const result = generateMazeGrid(7, 5, 7);

        expect(result).toHaveProperty('grid');
        expect(result).toHaveProperty('size');
        expect(result).toHaveProperty('entrance');
        expect(result).toHaveProperty('exit');
    });

    it('ensures odd dimensions', () => {
        const result = generateMazeGrid(6, 4, 8);
        const [w, h, l] = result.size;

        expect(w % 2).toBe(1);
        expect(h % 2).toBe(1);
        expect(l % 2).toBe(1);
    });

    it('creates entrance at z=0', () => {
        const result = generateMazeGrid(7, 5, 7);
        const [ex, ey, ez] = result.entrance;

        expect(ez).toBe(0);
        // Entrance should be carved (false = air)
        expect(result.grid[ex][ey][ez]).toBe(false);
    });

    it('creates exit at z=max', () => {
        const result = generateMazeGrid(7, 5, 7);
        const [, , mazeL] = result.size;
        const [ex, ey, ez] = result.exit;

        expect(ez).toBe(mazeL - 1);
        // Exit should be carved (false = air)
        expect(result.grid[ex][ey][ez]).toBe(false);
    });

    it('has floor and ceiling intact', () => {
        const result = generateMazeGrid(7, 5, 7);
        const [w, h, l] = result.size;

        // Check floor (y=0) and ceiling (y=max) are all walls
        for (let x = 0; x < w; x++) {
            for (let z = 0; z < l; z++) {
                expect(result.grid[x][0][z]).toBe(true); // Floor
                expect(result.grid[x][h-1][z]).toBe(true); // Ceiling
            }
        }
    });

    it('generates a solvable maze with connected passages', () => {
        const result = generateMazeGrid(7, 5, 7);
        const { grid, size } = result;
        const [w, h, l] = size;

        // Count carved cells (passages) - there should be some
        let passages = 0;
        for (let x = 0; x < w; x++) {
            for (let y = 1; y < h - 1; y++) {
                for (let z = 0; z < l; z++) {
                    if (!grid[x][y][z]) passages++;
                }
            }
        }

        // A proper maze should have a significant number of passages
        expect(passages).toBeGreaterThan(0);
    });
});

describe('gridToBlocks', () => {
    it('converts true cells to blocks', () => {
        const grid = createGrid(2, 2, 2, false);
        grid[0][0][0] = true;
        grid[1][1][1] = true;

        const blocks = gridToBlocks(grid, 'minecraft:stone');

        expect(blocks.length).toBe(2);
        expect(blocks).toContainEqual([0, 0, 0, 'minecraft:stone']);
        expect(blocks).toContainEqual([1, 1, 1, 'minecraft:stone']);
    });

    it('ignores false cells', () => {
        const grid = createGrid(2, 2, 2, false);
        const blocks = gridToBlocks(grid, 'minecraft:stone');

        expect(blocks.length).toBe(0);
    });

    it('uses the specified block type', () => {
        const grid = createGrid(1, 1, 1, true);
        const blocks = gridToBlocks(grid, 'minecraft:diamond_block');

        expect(blocks[0][3]).toBe('minecraft:diamond_block');
    });
});

describe('rotateBlocks', () => {
    const blocks = [[0, 0, 0, 'stone'], [2, 0, 0, 'stone'], [0, 0, 2, 'stone']];
    const size = [3, 1, 3];

    it('returns unchanged blocks for 0 rotation', () => {
        const rotated = rotateBlocks(blocks, size, 0);
        expect(rotated).toEqual(blocks);
    });

    it('rotates 90 degrees correctly', () => {
        const rotated = rotateBlocks([[0, 0, 0, 'stone']], [3, 1, 3], 90);
        // At 90°: nx = l - 1 - z = 2, nz = x = 0
        expect(rotated[0]).toEqual([2, 0, 0, 'stone']);
    });

    it('rotates 180 degrees correctly', () => {
        const rotated = rotateBlocks([[0, 0, 0, 'stone']], [3, 1, 3], 180);
        // At 180°: nx = w - 1 - x = 2, nz = l - 1 - z = 2
        expect(rotated[0]).toEqual([2, 0, 2, 'stone']);
    });

    it('rotates 270 degrees correctly', () => {
        const rotated = rotateBlocks([[0, 0, 0, 'stone']], [3, 1, 3], 270);
        // At 270°: nx = z = 0, nz = w - 1 - x = 2
        expect(rotated[0]).toEqual([0, 0, 2, 'stone']);
    });

    it('preserves y coordinate during rotation', () => {
        const rotated = rotateBlocks([[1, 5, 1, 'stone']], [3, 10, 3], 90);
        expect(rotated[0][1]).toBe(5);
    });
});

describe('getRotatedSize', () => {
    it('returns same size for 0 rotation', () => {
        expect(getRotatedSize([3, 5, 7], 0)).toEqual([3, 5, 7]);
    });

    it('returns same size for 180 rotation', () => {
        expect(getRotatedSize([3, 5, 7], 180)).toEqual([3, 5, 7]);
    });

    it('swaps width and length for 90 rotation', () => {
        expect(getRotatedSize([3, 5, 7], 90)).toEqual([7, 5, 3]);
    });

    it('swaps width and length for 270 rotation', () => {
        expect(getRotatedSize([3, 5, 7], 270)).toEqual([7, 5, 3]);
    });
});

describe('parseMazeOptions', () => {
    it('parses all options', () => {
        const options = parseMazeOptions('21 9 21 minecraft:stone_bricks');

        expect(options.width).toBe(21);
        expect(options.height).toBe(9);
        expect(options.length).toBe(21);
        expect(options.block).toBe('minecraft:stone_bricks');
    });

    it('handles partial options', () => {
        const options = parseMazeOptions('15 7');

        expect(options.width).toBe(15);
        expect(options.height).toBe(7);
        expect(options.length).toBeUndefined();
        expect(options.block).toBeUndefined();
    });

    it('handles empty string', () => {
        const options = parseMazeOptions('');

        expect(options.width).toBeUndefined();
        expect(options.height).toBeUndefined();
        expect(options.length).toBeUndefined();
        expect(options.block).toBeUndefined();
    });

    it('adds minecraft: prefix if missing', () => {
        const options = parseMazeOptions('15 7 15 cobblestone');
        expect(options.block).toBe('minecraft:cobblestone');
    });

    it('preserves full block ID if already prefixed', () => {
        const options = parseMazeOptions('15 7 15 minecraft:mossy_cobblestone');
        expect(options.block).toBe('minecraft:mossy_cobblestone');
    });

    it('handles extra whitespace', () => {
        const options = parseMazeOptions('  15   7   15  ');

        expect(options.width).toBe(15);
        expect(options.height).toBe(7);
        expect(options.length).toBe(15);
    });
});
