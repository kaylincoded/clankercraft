# Minecraft Urban Planner Skill

Architectural patterns for building realistic, visually rich urban structures. Derived from scanning dense urban cityscapes and iterating on builds in-game.

## Material Palette — Dense Urban Style

Dense urban builds use a deliberate gray/warm palette. Materials are grouped by architectural role:

### Primary Structure (60% of all blocks)
| Material | Count | Role |
|---|---|---|
| dirt | 218K | Foundation fill (y=-44 to -42) |
| polished_andesite | 198K | Primary facade, floors, modern surfaces |
| andesite | 143K | Secondary facade, rougher texture variant |
| smooth_sandstone | 124K | Warm-toned walls, middle-class buildings |
| dead_fire_coral_block | 119K | Dark gray accent, grime/weathering texture |
| stripped_birch_wood | 85K | Structural columns, load-bearing frames |

### Secondary Facade (25%)
| Material | Count | Role |
|---|---|---|
| stone_bricks | 83K | Traditional/older building walls |
| polished_diorite | 74K | Light-colored modern walls |
| mushroom_stem | 71K | Pale neutral walls (creative substitute for concrete) |
| diorite | 58K | Raw light stone, unfinished/cheap surfaces |
| iron_ore | 58K | Speckled texture noise — adds visual grit |
| gray_concrete | 53K | Modern brutalist surfaces |
| birch_planks | 48K | Interior floors, warm wooden accents |
| packed_mud | 41K | Streets, ground-level surfaces |

### Accent & Detail (15%)
| Material | Count | Role |
|---|---|---|
| white_terracotta | 35K | Clean wall accent |
| dead_tube/horn/bubble/brain_coral | 50K combined | Dark texture variants for weathering |
| smooth_quartz | 24K | Premium/modern building accent |
| granite | 17K | Warm accent, balcony details |
| sandstone variants | 29K | Warm-toned trim and detailing |
| trapdoors (6+ types) | 15K combined | Window shutters, AC units, facade depth |
| froglight (verdant/ochre) | 5.5K | Interior lighting, street lamps |

### Key Insight: Visual Noise Through Material Mixing
The city never uses a single material for a whole wall. Facades mix 3-5 materials from the same tonal family:
- **Gray family**: polished_andesite + andesite + dead_fire_coral + iron_ore + gray_concrete
- **Warm family**: smooth_sandstone + sandstone + granite + birch_planks + packed_mud
- **Light family**: polished_diorite + diorite + mushroom_stem + white_terracotta + smooth_quartz

This creates visual richness without explicit decoration — the texture variation reads as weathering, age, and patching.

## Construction Patterns

### Vertical Structure (typical building)
```
y=18-20:  Roof cap (polished_andesite solid + sandstone/smooth_stone trim)
y=17:     Top floor plate (mixed materials)
y=12-16:  Upper floors (5-block spacing, column + floor plate pattern)
y=-3-11:  Middle floors (same pattern repeating)
y=-38:    Ground floor plate
y=-39:    Ground level (street-facing)
y=-40:    Foundation slab (stone/cobbled_deepslate_slab)
y=-41:    Sub-foundation (stripped_birch_wood frame)
y=-44:    Bedrock-level fill (dirt)
```

### Floor Spacing: 5 blocks
Every floor occupies exactly 5 Y-levels:
- 1 block: floor plate (solid material row)
- 4 blocks: open space (walls on edges, air interior)

### Column System
- `stripped_birch_wood` runs as continuous vertical columns from foundation to roof
- Columns are placed at building corners and every ~8 blocks along walls
- Between columns: floor plates span horizontally in mixed materials

### Floor Plates
Each floor plate is a single Y-level filled across the building footprint. Material varies per floor — no two adjacent floors use the same mix. This creates the layered/stacked appearance visible from outside.

### Facade Rules
1. **Outer walls** are 1 block thick
2. **Material alternates** every 1-3 blocks horizontally — never a solid run of >3 of the same material
3. **Dead coral blocks** appear every 2-3 floors as dark "grime bands"
4. **Iron_ore** is scattered randomly (1-2 per floor) as visual noise
5. **Trapdoors** attached to walls simulate AC units, shutters, and balcony railings

### Street Level
- Streets: `packed_mud` or `dirt_path` (3-4 blocks wide)
- Sidewalks: `smooth_stone` or `cobbled_deepslate_slab`
- Street-level facades use darker materials (deepslate, cobblestone) for shadow depth
- `scaffolding` and `ladder` appear on some buildings — construction/fire-escape aesthetic

## Building Typologies (from scan data)

### Type A: Residential Tower (most common)
- Footprint: 8-12 blocks square
- Height: 50-70 blocks (10-14 floors)
- Palette: polished_andesite + andesite + dead_coral + iron_ore
- Features: stripped_birch_wood columns, trapdoor "balconies"

### Type B: Commercial/Mixed-Use
- Footprint: 12-16 blocks, often rectangular
- Height: 30-50 blocks (6-10 floors)
- Palette: smooth_sandstone + sandstone + birch_planks + granite
- Features: wider windows (glass panes), signs, ground-floor shops

### Type C: Modern/Premium
- Footprint: varies
- Height: varies
- Palette: polished_diorite + smooth_quartz + gray_concrete + white_terracotta
- Features: cleaner lines, fewer texture variations, froglight lighting

## Rules for Building Dense Urban Structures

1. NEVER use a single material for an entire wall. Mix 3-5 materials from one tonal family.
2. Use 5-block floor spacing (1 plate + 4 air).
3. Place stripped_birch_wood columns at corners and every 8 blocks.
4. Vary floor plate materials — each floor should use a different primary material.
5. Add dead_fire_coral_block as dark "grime bands" every 2-3 floors.
6. Scatter iron_ore (1-2 per floor) as visual noise on facades.
7. Use trapdoors on exterior walls for depth — they represent AC units, shutters, railings.
8. Streets are packed_mud/dirt_path; sidewalks are smooth_stone/deepslate_slab.
9. Ground floors should be darker (deepslate, cobblestone) than upper floors.
10. Froglight for interior lighting and street lamps.

## Rules for Small Commercial Facades (Shops, Storefronts)

### Glass
11. NEVER use `glass_pane` on facades — it creates thin, disconnected-looking windows with awkward gaps. ALWAYS use full `glass` blocks.
    - Evidence: glass_pane rendered as thin lines with visible gaps between adjacent panes on a 5-shop strip. Replacing with `glass` blocks made windows look solid and intentional.

### Facade Depth
12. NEVER build a perfectly flat facade (all blocks at same Z). Add depth by:
    - Recessing windows 1 block back from the wall plane (place glass at z+1, wall frame at z)
    - Adding stair blocks as window sills (bottom) and lintels (top)
    - Placing trapdoors as shutters flanking windows
    - Using slabs or stairs as cornice projections at roofline (overhang at z-1)
    - Evidence: shops with every block flush at the same Z look like flat textures painted on a wall. Real buildings have shadow lines from depth variation.

### Roofline / Cornice
13. ALWAYS cap buildings with a projecting cornice — stairs or slabs extending 1 block outward from the facade at the roofline. This creates a strong shadow line and a finished "top edge."
    - Evidence: Shops ended abruptly with flat slab rows or just stopped. No visual termination at the top.

### Street-Level Treatment
14. ALWAYS differentiate the ground floor from upper floors:
    - Use a darker or heavier material for the base (deepslate, stone_bricks, dark_oak_planks)
    - Add an awning/overhang (1-2 block projection using slabs, stairs, or wool) above the storefront
    - Make storefront windows taller (2-3 blocks) than upper floor windows
    - Evidence: Shops lacked awnings or overhangs, making the ground floor indistinguishable from upper stories.

### Window Framing
15. ALWAYS frame windows with trim material — at minimum a sill (stair/slab below) and header (stair/slab above). Use a contrasting but complementary material to the wall.
    - Window sills (below): use `half=top` so the flat ledge faces upward — creates a shelf. Without `half=top`, stairs read as walkable steps, not a ledge.
    - Window lintels (above): use `half=top` so the flat part faces downward — creates a header.
    - Good: smooth_sandstone wall + granite stair sills + granite stair lintels
    - Evidence: Default stairs as sills looked like steps leading to the window. Flipping to `half=top` makes them read as architectural ledges.
16. NEVER use stair blocks as window sills at ground level (player-reachable height). Stairs at ground level read as functional navigation (walkable steps), not decorative trim — players will try to walk up to the window. Only use stair sills/lintels at heights players can't reach (y >= 3 blocks above ground). For ground-level window trim, use slabs, trapdoors, or full blocks instead.
    - Evidence: stairs at ground level in front of windows looked like steps leading to the window, creating confusing navigation cues.

## Scanner Tool

The Go world scanner at `tools/world-scanner/` can scan any local world save:
```bash
cd tools/world-scanner
go build -o world-scanner .
./world-scanner "/path/to/world" --bounds minX minY minZ maxX maxY maxZ --json output.json
```
Scans 15M blocks in ~1 second. Use this to analyze reference builds before attempting to reproduce them.
