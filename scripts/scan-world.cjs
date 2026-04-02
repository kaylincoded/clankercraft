#!/usr/bin/env node
// Scan a local Minecraft world save and output block data
// Usage: node scan-world.cjs <world-path> <x1> <y1> <z1> <x2> <y2> <z2> [--materials-only] [--slice y]

const AnvilFactory = require('prismarine-provider-anvil').Anvil;
const Anvil = AnvilFactory('1.21');
const { Vec3 } = require('vec3');
const mcData = require('minecraft-data')('1.21.1');

async function scanWorld(worldPath, x1, y1, z1, x2, y2, z2) {
  const regionPath = worldPath + '/region';
  const anvil = new Anvil(regionPath);

  const minX = Math.min(x1, x2), maxX = Math.max(x1, x2);
  const minY = Math.min(y1, y2), maxY = Math.max(y1, y2);
  const minZ = Math.min(z1, z2), maxZ = Math.max(z1, z2);

  const volume = (maxX - minX + 1) * (maxY - minY + 1) * (maxZ - minZ + 1);
  if (volume > 100000) {
    console.error(`Volume too large: ${volume} (max 100000)`);
    process.exit(1);
  }

  const chunkCache = new Map();

  async function getBlock(x, y, z) {
    const chunkX = Math.floor(x / 16);
    const chunkZ = Math.floor(z / 16);
    const key = `${chunkX},${chunkZ}`;

    if (!chunkCache.has(key)) {
      try {
        const chunk = await anvil.load(chunkX, chunkZ);
        chunkCache.set(key, chunk);
      } catch (e) {
        chunkCache.set(key, null);
      }
    }

    const chunk = chunkCache.get(key);
    if (!chunk) return null;

    try {
      const stateId = chunk.getBlockStateId(new Vec3(x & 15, y, z & 15));
      if (stateId === 0) return null; // air
      const block = mcData.blocksByStateId[stateId];
      return block ? block.name : `unknown_${stateId}`;
    } catch (e) {
      return null;
    }
  }

  const blocks = {};
  const materialCounts = {};

  for (let y = minY; y <= maxY; y++) {
    for (let x = minX; x <= maxX; x++) {
      for (let z = minZ; z <= maxZ; z++) {
        const name = await getBlock(x, y, z);
        if (name) {
          blocks[`${x},${y},${z}`] = name;
          materialCounts[name] = (materialCounts[name] || 0) + 1;
        }
      }
    }
  }

  return { blocks, materialCounts, volume };
}

async function main() {
  const args = process.argv.slice(2);
  if (args.length < 7) {
    console.log('Usage: node scan-world.cjs <world-path> <x1> <y1> <z1> <x2> <y2> <z2> [--materials-only] [--slice <y>]');
    process.exit(1);
  }

  const [worldPath, x1, y1, z1, x2, y2, z2] = args;
  const materialsOnly = args.includes('--materials-only');
  const sliceIdx = args.indexOf('--slice');

  const result = await scanWorld(worldPath, +x1, +y1, +z1, +x2, +y2, +z2);
  const blockCount = Object.keys(result.blocks).length;

  if (sliceIdx !== -1 && args[sliceIdx + 1]) {
    // Print a 2D grid at the given Y level
    const sliceY = +args[sliceIdx + 1];
    const mnX = Math.min(+x1, +x2), mxX = Math.max(+x1, +x2);
    const mnZ = Math.min(+z1, +z2), mxZ = Math.max(+z1, +z2);
    console.log(`Slice at y=${sliceY}:`);
    for (let z = mnZ; z <= mxZ; z++) {
      let row = '';
      for (let x = mnX; x <= mxX; x++) {
        const b = result.blocks[`${x},${sliceY},${z}`];
        row += b ? b.substring(0, 3).padEnd(4) : '.   ';
      }
      console.log(`z=${z}: ${row}`);
    }
  } else if (materialsOnly) {
    const sorted = Object.entries(result.materialCounts).sort((a, b) => b[1] - a[1]);
    console.log(`Materials (${blockCount} non-air blocks in ${result.volume} total):`);
    sorted.forEach(([name, count]) => console.log(`  ${count}x ${name}`));
  } else {
    Object.entries(result.blocks).forEach(([pos, name]) => {
      console.log(`${pos}:${name}`);
    });
    console.log(`\n--- Materials summary (${blockCount} blocks) ---`);
    const sorted = Object.entries(result.materialCounts).sort((a, b) => b[1] - a[1]);
    sorted.forEach(([name, count]) => console.log(`  ${count}x ${name}`));
  }
}

main().catch(e => { console.error(e); process.exit(1); });
