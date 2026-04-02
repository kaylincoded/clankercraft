import { z } from "zod";
import mineflayer from 'mineflayer';
import pathfinderPkg from 'mineflayer-pathfinder';
const { goals } = pathfinderPkg;
import { Vec3 } from 'vec3';
import minecraftData from 'minecraft-data';
import { ToolFactory } from '../tool-factory.js';
import { log } from '../logger.js';
import { coerceCoordinates } from './coordinate-utils.js';

type FaceDirection = 'up' | 'down' | 'north' | 'south' | 'east' | 'west';

interface FaceOption {
  direction: string;
  vector: Vec3;
}

export function registerBlockTools(factory: ToolFactory, getBot: () => mineflayer.Bot): void {
  factory.registerTool(
    "place-block",
    "Place a block at the specified position",
    {
      x: z.coerce.number().describe("X coordinate"),
      y: z.coerce.number().describe("Y coordinate"),
      z: z.coerce.number().describe("Z coordinate"),
      faceDirection: z.enum(['up', 'down', 'north', 'south', 'east', 'west']).optional().describe("Direction to place against (default: 'down')")
    },
    async ({ x, y, z, faceDirection = 'down' }: { x: number, y: number, z: number, faceDirection?: FaceDirection }) => {
      ({ x, y, z } = coerceCoordinates(x, y, z));

      const bot = getBot();
      const placePos = new Vec3(x, y, z);
      const blockAtPos = bot.blockAt(placePos);

      if (blockAtPos && blockAtPos.name !== 'air') {
        return factory.createResponse(`There's already a block (${blockAtPos.name}) at (${x}, ${y}, ${z})`);
      }

      const possibleFaces: FaceOption[] = [
        { direction: 'down', vector: new Vec3(0, -1, 0) },
        { direction: 'north', vector: new Vec3(0, 0, -1) },
        { direction: 'south', vector: new Vec3(0, 0, 1) },
        { direction: 'east', vector: new Vec3(1, 0, 0) },
        { direction: 'west', vector: new Vec3(-1, 0, 0) },
        { direction: 'up', vector: new Vec3(0, 1, 0) }
      ];

      if (faceDirection !== 'down') {
        const specificFace = possibleFaces.find(face => face.direction === faceDirection);
        if (specificFace) {
          possibleFaces.unshift(possibleFaces.splice(possibleFaces.indexOf(specificFace), 1)[0]);
        }
      }

      const debugInfo: string[] = [];

      for (const face of possibleFaces) {
        const referencePos = placePos.plus(face.vector);
        const referenceBlock = bot.blockAt(referencePos);

        if (!referenceBlock || referenceBlock.name === 'air') {
          debugInfo.push(`${face.direction}: air/null at (${referencePos.x},${referencePos.y},${referencePos.z})`);
          continue;
        }

        const canSee = bot.canSeeBlock(referenceBlock);
        debugInfo.push(`${face.direction}: ${referenceBlock.name} at (${referencePos.x},${referencePos.y},${referencePos.z}), canSee=${canSee}`);

        if (!canSee) {
          try {
            if (bot.game.gameMode === 'creative') {
              await bot.creative.flyTo(new Vec3(referencePos.x, referencePos.y + 1, referencePos.z));
            } else {
              const goal = new goals.GoalNear(referencePos.x, referencePos.y, referencePos.z, 2);
              await bot.pathfinder.goto(goal);
            }
            debugInfo.push(`  moved closer, canSee now=${bot.canSeeBlock(referenceBlock)}`);
          } catch (moveError) {
            debugInfo.push(`  move failed: ${moveError}`);
          }
        }

        await bot.lookAt(placePos, true);

        try {
          await bot.placeBlock(referenceBlock, face.vector.scaled(-1));
          return factory.createResponse(`Placed block at (${x}, ${y}, ${z}) using ${face.direction} face`);
        } catch (placeError) {
          debugInfo.push(`  placeBlock failed: ${placeError}`);
          continue;
        }
      }

      return factory.createResponse(`Failed to place block at (${x}, ${y}, ${z}). Debug:\n${debugInfo.join('\n')}`);
    }
  );

  factory.registerTool(
    "dig-block",
    "Dig a block at the specified position",
    {
      x: z.coerce.number().describe("X coordinate"),
      y: z.coerce.number().describe("Y coordinate"),
      z: z.coerce.number().describe("Z coordinate"),
    },
    async ({ x, y, z }) => {
      ({ x, y, z } = coerceCoordinates(x, y, z));

      const bot = getBot();
      const blockPos = new Vec3(x, y, z);
      const block = bot.blockAt(blockPos);

      if (!block || block.name === 'air') {
        return factory.createResponse(`No block found at position (${x}, ${y}, ${z})`);
      }

      if (!bot.canDigBlock(block) || !bot.canSeeBlock(block)) {
        try {
          if (bot.game.gameMode === 'creative') {
            await bot.creative.flyTo(new Vec3(x, y + 1, z));
          } else {
            const goal = new goals.GoalNear(x, y, z, 2);
            await bot.pathfinder.goto(goal);
          }
        } catch (moveError) {
          log('warn', `Failed to move near block for digging: ${moveError}`);
        }
      }

      await bot.dig(block);
      return factory.createResponse(`Dug ${block.name} at (${x}, ${y}, ${z})`);
    }
  );

  factory.registerTool(
    "get-block-info",
    "Get information about a block at the specified position",
    {
      x: z.coerce.number().describe("X coordinate"),
      y: z.coerce.number().describe("Y coordinate"),
      z: z.coerce.number().describe("Z coordinate"),
    },
    async ({ x, y, z }) => {
      ({ x, y, z } = coerceCoordinates(x, y, z));

      const bot = getBot();
      const blockPos = new Vec3(x, y, z);
      const block = bot.blockAt(blockPos);

      if (!block) {
        return factory.createResponse(`No block information found at position (${x}, ${y}, ${z})`);
      }

      return factory.createResponse(`Found ${block.name} (type: ${block.type}) at position (${block.position.x}, ${block.position.y}, ${block.position.z})`);
    }
  );

  factory.registerTool(
    "read-sign",
    "Read the text on a sign at the specified position",
    {
      x: z.coerce.number().describe("X coordinate"),
      y: z.coerce.number().describe("Y coordinate"),
      z: z.coerce.number().describe("Z coordinate"),
    },
    async ({ x, y, z }) => {
      ({ x, y, z } = coerceCoordinates(x, y, z));

      const bot = getBot();
      const blockPos = new Vec3(x, y, z);
      const block = bot.blockAt(blockPos);

      if (!block) {
        return factory.createResponse(`No block found at position (${x}, ${y}, ${z})`);
      }

      if (!block.name.includes('sign')) {
        return factory.createResponse(`Block at (${x}, ${y}, ${z}) is ${block.name}, not a sign`);
      }

      const texts = (block as any).getSignText();
      if (!texts || texts.every((t: string) => !t)) {
        return factory.createResponse(`Sign at (${x}, ${y}, ${z}) is blank`);
      }

      const parts: string[] = [];
      if (texts[0]) parts.push(`Front: ${texts[0]}`);
      if (texts[1]) parts.push(`Back: ${texts[1]}`);
      return factory.createResponse(`Sign at (${x}, ${y}, ${z}):\n${parts.join('\n')}`);
    }
  );

  factory.registerTool(
    "find-signs",
    "Find all signs within a given distance and read their text",
    {
      maxDistance: z.coerce.number().finite().optional().describe("Maximum search distance (default: 50)")
    },
    async ({ maxDistance = 50 }) => {
      const bot = getBot();
      const mcData = minecraftData(bot.version);
      const signBlockIds = Object.values(mcData.blocksByName)
        .filter((b: any) => b.name.includes('sign'))
        .map((b: any) => b.id);

      const signs = bot.findBlocks({
        matching: signBlockIds,
        maxDistance,
        count: 50
      });

      if (signs.length === 0) {
        return factory.createResponse(`No signs found within ${maxDistance} blocks`);
      }

      const results = signs.map(pos => {
        const block = bot.blockAt(pos);
        if (!block) return null;
        const texts = (block as any).getSignText();
        const front = texts?.[0] || '';
        const back = texts?.[1] || '';
        const text = [front, back].filter(Boolean).join(' | ') || '(blank)';
        return `  ${block.name} at (${pos.x}, ${pos.y}, ${pos.z}): ${text}`;
      }).filter(Boolean);

      return factory.createResponse(`Found ${results.length} signs:\n${results.join('\n')}`);
    }
  );

  factory.registerTool(
    "scan-area",
    "Scan a rectangular area and return all non-air blocks. Max 10000 blocks per scan.",
    {
      x1: z.coerce.number().describe("First corner X"),
      y1: z.coerce.number().describe("First corner Y"),
      z1: z.coerce.number().describe("First corner Z"),
      x2: z.coerce.number().describe("Second corner X"),
      y2: z.coerce.number().describe("Second corner Y"),
      z2: z.coerce.number().describe("Second corner Z"),
      includeAir: z.boolean().optional().describe("Include air blocks in output (default: false)")
    },
    async ({ x1, y1, z1, x2, y2, z2, includeAir = false }) => {
      ({ x: x1, y: y1, z: z1 } = coerceCoordinates(x1, y1, z1));
      ({ x: x2, y: y2, z: z2 } = coerceCoordinates(x2, y2, z2));

      const minX = Math.min(x1, x2), maxX = Math.max(x1, x2);
      const minY = Math.min(y1, y2), maxY = Math.max(y1, y2);
      const minZ = Math.min(z1, z2), maxZ = Math.max(z1, z2);

      const volume = (maxX - minX + 1) * (maxY - minY + 1) * (maxZ - minZ + 1);
      if (volume > 10000) {
        return factory.createResponse(`Area too large: ${volume} blocks (max 10000). Reduce the scan area.`);
      }

      const bot = getBot();
      const blocks: string[] = [];

      for (let y = minY; y <= maxY; y++) {
        for (let x = minX; x <= maxX; x++) {
          for (let zz = minZ; zz <= maxZ; zz++) {
            const block = bot.blockAt(new Vec3(x, y, zz));
            if (block && (includeAir || block.name !== 'air')) {
              blocks.push(`${x},${y},${zz}:${block.name}`);
            }
          }
        }
      }

      if (blocks.length === 0) {
        return factory.createResponse(`No ${includeAir ? '' : 'non-air '}blocks found in area (${minX},${minY},${minZ}) to (${maxX},${maxY},${maxZ})`);
      }

      return factory.createResponse(`Scanned ${volume} blocks, found ${blocks.length} ${includeAir ? '' : 'non-air '}blocks:\n${blocks.join('\n')}`);
    }
  );

  factory.registerTool(
    "find-block",
    "Find the nearest block of a specific type",
    {
      blockType: z.string().describe("Type of block to find"),
      maxDistance: z.coerce.number().finite().optional().describe("Maximum search distance (default: 16)")
    },
    async ({ blockType, maxDistance = 16 }) => {
      const bot = getBot();
      const mcData = minecraftData(bot.version);
      const blocksByName = mcData.blocksByName;

      if (!blocksByName[blockType]) {
        return factory.createResponse(`Unknown block type: ${blockType}`);
      }

      const blockId = blocksByName[blockType].id;

      const block = bot.findBlock({
        matching: blockId,
        maxDistance: maxDistance
      });

      if (!block) {
        return factory.createResponse(`No ${blockType} found within ${maxDistance} blocks`);
      }

      return factory.createResponse(`Found ${blockType} at position (${block.position.x}, ${block.position.y}, ${block.position.z})`);
    }
  );
}
