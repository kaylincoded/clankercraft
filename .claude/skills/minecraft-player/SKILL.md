# Minecraft Player Skill

Learned patterns for using the Minecraft MCP tools effectively. These are injected context — imperative rules derived from actual execution experience.

## Building — Use Server Commands, Not place-block

- ALWAYS use `/fill` via `send-chat` for building structures. It places blocks instantly with no reach limit, no chunk loading issues, and can fill rectangular regions in one command.
  - Syntax: `send-chat("/fill x1 y1 z1 x2 y2 z2 block_type")`
  - Example wall: `send-chat("/fill 982 107 335 986 110 335 stone")` — builds a 5x4 wall instantly.
  - Evidence: `/fill` built a 5x4 wall (20 blocks) in one command. `place-block` required tp + chunk loading + 1-block reach = ~10 seconds per block.
- NEVER send multiple `/fill` or `/setblock` commands in parallel. Minecraft chat has rate limiting — only the first command executes, the rest are silently dropped.
  - Evidence: 4 parallel `/fill` commands for a box — only the first wall built. The other 3 were lost.
- Use `/setblock x y z block_type` via `send-chat` for placing individual blocks at precise positions.
- Use `/fill x1 y1 z1 x2 y2 z2 air` to clear/demolish regions.
- WorldEdit is also available (`//set`, `//walls`, etc.) for complex operations.
- NEVER use the `place-block` MCP tool for building structures. It has a 1-block reach limit, requires chunk loading after tp, and fails silently from >1 block away. Only useful for single-block interactions where you're already adjacent.
  - Evidence: `place-block` fails with "blockUpdate event did not fire within timeout of 5000ms" from >1 block away. After `/tp`, chunks need a `scan-area` call to load before `place-block` works.

## Building Strategies

- ALWAYS plan the full build as a coordinate blueprint BEFORE executing any commands. Design each wall, floor, and detail as a list of `/fill` commands with exact coordinates and materials.
  - Evidence: Improvising commands one-by-one produced a featureless box with missing blocks and no architectural detail.
- Build structures layer by layer (bottom to top) using `/fill` for each rectangular section.
- NEVER use `air` for doorways. Use actual door blocks with blockstates: `/setblock X Y Z oak_door[half=lower,facing=north]` + `/setblock X Y+1 Z oak_door[half=upper,facing=north]`. For double doors, set `hinge=right` on the second door.
  - Match door material to building style: oak (warm), spruce (rustic), iron (modern), dark_oak (upscale), warped (nautical/teal).
  - Iron doors require a redstone signal to open. Pressure plates/buttons must be DIRECTLY ADJACENT to the door block (sharing a face) and attached to an OPAQUE block (not glass — glass is transparent to redstone). For storefronts, prefer regular doors (dark_oak for modern look) over iron to avoid redstone complexity.
  - Inside: pressure plate on the floor directly behind the door (z+1 at floor level).
  - Outside: stone button on an opaque wall block adjacent to the door, NOT on glass.
  - NEVER place pressure plates on a separate Z-plane (like z=334) expecting them to power doors at z=335 — the signal won't reach diagonally.
  - Evidence: plates on the sidewalk diagonal to the door on another Z-plane won't reach — the signal doesn't travel diagonally. Buttons on glass won't transmit power either.
  - When the floor is higher than the street, add a stair block at street level facing the door: `/setblock X Y Z material_stairs[facing=south]`. Match stair material to the building style.
  - Evidence: Doors placed at y=108 with street at y=106 were unreachable. Stairs at y=107 facing the door fix the step-up. Air gaps in facades look broken — real doors are functional and visually correct.
- NEVER use `glass_pane` for windows on facades — it creates thin, disconnected-looking windows with visible gaps. ALWAYS use full `glass` blocks.
  - Evidence: glass_pane on a shop strip looked like thin lines with awkward gaps between adjacent panes.
- For walls: use `/fill` with a 1-block-thick region (e.g., same z1 and z2 for a north-south wall).
- Add architectural depth: use stone_brick_stairs as trim, vary wall depth with 1-block setbacks, mix materials per floor.
- ALWAYS verify each wall with `scan-area` before moving to the next — catch rate-limit drops early.
- Use `scan-area` before building to check the ground level and ensure the area is clear.
- Ask the user for in-game screenshots to verify aesthetics — scan-area confirms structure but not appearance.

## Visualization

- Prismarine viewer is NOT reliable — it crashes on unknown entities (glow_squid, item) because it doesn't support MC 1.21.11. Do not depend on it.
- Use `scan-area` for structural verification. Scan horizontal slices to check floor plans, vertical slices to check wall heights.
- Ask the user for visual feedback on aesthetics — scan-area can verify structure but not appearance.

## Inventory & Equipment

- ALWAYS use exact full item names from `list-inventory` output when calling `equip-item` or `find-item`.
  - Fixed in MCP server (exact match preferred over substring), but still use full names as a habit.
- For building with `/fill` and `/setblock`, inventory doesn't matter — these are server commands that create blocks directly.

## Movement & Navigation

- On creative servers, use `/tp` via `send-chat` for movement: `send-chat("/tp BOT_USERNAME X Y Z")` (bot username is configured in `.env` as `MC_USERNAME`). Both `fly-to` and `move-to-position` are unreliable.
  - Evidence: fly-to timed out at 20s for 18-block distance. move-to-position timed out at 60s without moving.
- After `/tp`, chunks may not be loaded. Call `scan-area` or `get-block-info` to force chunk loading before using `place-block` (not needed for `/fill` or `/setblock`).

## Coordinate System

- Y is vertical (up/down). Higher Y = higher altitude.
- The bot's position Y is where its feet are. Ground blocks are typically at Y-1 relative to bot position.
- `scan-area` max is 10,000 blocks per call. For large areas, scan in chunks.
