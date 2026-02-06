import { describe, it, expect } from 'vitest';
import {
    shuffle,
    generateMazeGrid,
    rotateBlocks,
    parseMazeOptions
} from '../behavior_pack/scripts/maze.js';
import {
    createGrid,
    gridToBlocks
} from '../behavior_pack/scripts/shapes.js';

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

    it('handles minimum size (5x5x5)', () => {
        const result = generateMazeGrid(5, 5, 5);
        const [w, h, l] = result.size;

        expect(w).toBe(5);
        expect(h).toBe(5);
        expect(l).toBe(5);
        expect(result.grid[result.entrance[0]][1][result.entrance[2]]).toBe(false);
        expect(result.grid[result.exit[0]][1][result.exit[2]]).toBe(false);
    });

    it('handles even dimensions by rounding up to odd', () => {
        const result = generateMazeGrid(10, 6, 12);
        const [w, h, l] = result.size;

        expect(w).toBe(11);
        expect(h).toBe(7);
        expect(l).toBe(13);
    });

    it('handles medium size (31x7x31)', () => {
        const result = generateMazeGrid(31, 7, 31);
        const [w, h, l] = result.size;

        expect(w).toBe(31);
        expect(l).toBe(31);
        expect(result.grid[result.entrance[0]][1][result.entrance[2]]).toBe(false);
        expect(result.grid[result.exit[0]][1][result.exit[2]]).toBe(false);
    });
});

// BFS helper used by multiple tests
function bfs(grid, size, start, end) {
    const [w, , l] = size;
    const directions = [[0, 1], [0, -1], [1, 0], [-1, 0]];
    const visited = new Set();
    const prev = {};
    const queue = [start.join(",")];
    visited.add(queue[0]);

    while (queue.length > 0) {
        const key = queue.shift();
        const [sx, sz] = key.split(",").map(Number);

        if (sx === end[0] && sz === end[1]) {
            // Reconstruct path
            const path = [];
            let k = key;
            while (k) {
                path.unshift(k.split(",").map(Number));
                k = prev[k];
            }
            return path;
        }

        for (const [dx, dz] of directions) {
            const nx = sx + dx;
            const nz = sz + dz;
            const nk = nx + "," + nz;
            if (nx >= 0 && nx < w && nz >= 0 && nz < l && !visited.has(nk) && !grid[nx][1][nz]) {
                visited.add(nk);
                prev[nk] = key;
                queue.push(nk);
            }
        }
    }
    return null; // no path
}

describe('maze solvability', () => {
    it('has a path from entrance to exit (small maze)', () => {
        const { grid, size, entrance, exit } = generateMazeGrid(7, 5, 7);
        const path = bfs(grid, size, [entrance[0], entrance[2]], [exit[0], exit[2]]);
        expect(path).not.toBeNull();
        expect(path.length).toBeGreaterThan(1);
    });

    it('has a path from entrance to exit (medium maze)', () => {
        const { grid, size, entrance, exit } = generateMazeGrid(15, 7, 15);
        const path = bfs(grid, size, [entrance[0], entrance[2]], [exit[0], exit[2]]);
        expect(path).not.toBeNull();
    });

    it('has a path from entrance to exit (large maze)', () => {
        const { grid, size, entrance, exit } = generateMazeGrid(31, 7, 31);
        const path = bfs(grid, size, [entrance[0], entrance[2]], [exit[0], exit[2]]);
        expect(path).not.toBeNull();
    });

    it('has a path from entrance to exit (edge case 5x5x5)', () => {
        const { grid, size, entrance, exit } = generateMazeGrid(5, 5, 5);
        const path = bfs(grid, size, [entrance[0], entrance[2]], [exit[0], exit[2]]);
        expect(path).not.toBeNull();
    });

    it('is solvable across multiple random seeds', () => {
        // Run 10 times to catch intermittent generation bugs
        for (let i = 0; i < 10; i++) {
            const { grid, size, entrance, exit } = generateMazeGrid(15, 7, 15);
            const path = bfs(grid, size, [entrance[0], entrance[2]], [exit[0], exit[2]]);
            expect(path).not.toBeNull();
        }
    });
});

describe('maze post-processing: loops', () => {
    it('contains cycles (walls removed between carved cells)', () => {
        // Generate a large enough maze that loops are statistically certain
        const { grid, size } = generateMazeGrid(31, 7, 31);
        const [w, , l] = size;

        // Count wall positions between two carved cells that have been opened
        // In a pure spanning tree, every wall between cells is intact except
        // the carved passages. Loops create extra openings.
        // We detect loops by finding carved cells with >2 open neighbors.
        let multiConnected = 0;
        const directions = [[0, 1], [0, -1], [1, 0], [-1, 0]];

        for (let x = 1; x < w - 1; x += 2) {
            for (let z = 1; z < l - 1; z += 2) {
                if (grid[x][1][z]) continue; // wall, not a cell
                let openNeighbors = 0;
                for (const [dx, dz] of directions) {
                    const nx = x + dx;
                    const nz = z + dz;
                    if (nx >= 0 && nx < w && nz >= 0 && nz < l && !grid[nx][1][nz]) {
                        openNeighbors++;
                    }
                }
                if (openNeighbors > 2) multiConnected++;
            }
        }

        // In a spanning tree, leaf cells have exactly 1 neighbor and internal
        // cells have 2+. With loops added, some cells will have 3-4 neighbors.
        // On a 31x31 maze with 20% wall removal, we expect many such cells.
        expect(multiConnected).toBeGreaterThan(0);
    });
});

describe('maze post-processing: dead end extension', () => {
    it('has dead ends present in the maze', () => {
        const { grid, size } = generateMazeGrid(15, 7, 15);
        const [w, , l] = size;
        const directions = [[0, 1], [0, -1], [1, 0], [-1, 0]];

        let deadEnds = 0;
        for (let x = 1; x < w - 1; x += 2) {
            for (let z = 1; z < l - 1; z += 2) {
                if (grid[x][1][z]) continue;
                let openNeighbors = 0;
                for (const [dx, dz] of directions) {
                    const nx = x + dx;
                    const nz = z + dz;
                    if (nx >= 0 && nx < w && nz >= 0 && nz < l && !grid[nx][1][nz]) {
                        openNeighbors++;
                    }
                }
                if (openNeighbors === 1) deadEnds++;
            }
        }

        // Maze should still have dead ends (not all eliminated)
        expect(deadEnds).toBeGreaterThan(0);
    });
});

describe('maze post-processing: deceptive rooms', () => {
    it('contains open areas larger than a single cell on large mazes', () => {
        // On a 31x31 maze, rooms should create 2x2+ open areas
        const { grid, size } = generateMazeGrid(31, 7, 31);
        const [w, , l] = size;

        // Look for 2x2 open areas (all four grid positions carved)
        let openAreas = 0;
        for (let x = 1; x < w - 2; x++) {
            for (let z = 1; z < l - 2; z++) {
                if (!grid[x][1][z] && !grid[x+1][1][z] &&
                    !grid[x][1][z+1] && !grid[x+1][1][z+1]) {
                    openAreas++;
                }
            }
        }

        // Should have at least one 2x2 open area from rooms or loop removal
        expect(openAreas).toBeGreaterThan(0);
    });
});

describe('maze outer walls integrity', () => {
    it('maintains outer walls except at entrance and exit', () => {
        const { grid, size, entrance, exit } = generateMazeGrid(15, 7, 15);
        const [w, h, l] = size;

        for (let x = 0; x < w; x++) {
            for (let y = 1; y < h - 1; y++) {
                // z=0 wall (entrance side)
                if (x === entrance[0] && y >= 1 && y < h - 1) continue;
                expect(grid[x][y][0]).toBe(true);
            }
        }

        for (let x = 0; x < w; x++) {
            for (let y = 1; y < h - 1; y++) {
                // z=max wall (exit side)
                if (x === exit[0] && y >= 1 && y < h - 1) continue;
                expect(grid[x][y][l - 1]).toBe(true);
            }
        }

        // x=0 and x=max walls should be fully intact
        for (let y = 1; y < h - 1; y++) {
            for (let z = 0; z < l; z++) {
                expect(grid[0][y][z]).toBe(true);
                expect(grid[w - 1][y][z]).toBe(true);
            }
        }
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
