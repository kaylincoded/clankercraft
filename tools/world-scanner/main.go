package main

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Tnze/go-mc/save"
	"github.com/Tnze/go-mc/save/region"
)

type ScanResult struct {
	Materials map[string]int64  `json:"materials"`
	Bounds    Bounds            `json:"bounds"`
	Blocks    map[string]string `json:"blocks,omitempty"` // "x,y,z" -> "block_name"
}

type Bounds struct {
	MinX, MaxX, MinY, MaxY, MinZ, MaxZ int
}

func getBlockFromSection(sec *save.Section, x, y, z int) string {
	palette := sec.BlockStates.Palette
	if len(palette) == 0 {
		return "air"
	}
	if len(palette) == 1 {
		return palette[0].Name
	}

	data := sec.BlockStates.Data
	if len(data) == 0 {
		return palette[0].Name
	}

	bitsPerEntry := max(4, bits.Len(uint(len(palette)-1)))
	entriesPerLong := 64 / bitsPerEntry
	mask := uint64((1 << bitsPerEntry) - 1)

	index := y*16*16 + z*16 + x
	longIndex := index / entriesPerLong
	bitOffset := (index % entriesPerLong) * bitsPerEntry

	if longIndex >= len(data) {
		return "air"
	}

	paletteIndex := int((data[longIndex] >> uint(bitOffset)) & mask)
	if paletteIndex >= len(palette) {
		return "air"
	}

	return palette[paletteIndex].Name
}

func scanRegionFile(path string, bounds Bounds, mu *sync.Mutex, result *ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	r, err := region.Open(path)
	if err != nil {
		return
	}
	defer r.Close()

	// Parse region coords from filename: r.X.Z.mca
	base := filepath.Base(path)
	var rx, rz int
	fmt.Sscanf(base, "r.%d.%d.mca", &rx, &rz)

	localMaterials := make(map[string]int64)

	for cx := 0; cx < 32; cx++ {
		for cz := 0; cz < 32; cz++ {
			if !r.ExistSector(cx, cz) {
				continue
			}

			data, err := r.ReadSector(cx, cz)
			if err != nil {
				continue
			}

			var chunk save.Chunk
			if err := chunk.Load(data); err != nil {
				continue
			}

			chunkWorldX := (rx*32 + cx) * 16
			chunkWorldZ := (rz*32 + cz) * 16

			// Quick bounds check at chunk level
			if chunkWorldX+15 < bounds.MinX || chunkWorldX > bounds.MaxX ||
				chunkWorldZ+15 < bounds.MinZ || chunkWorldZ > bounds.MaxZ {
				continue
			}

			for _, sec := range chunk.Sections {
				secWorldY := int(sec.Y) * 16

				if secWorldY+15 < bounds.MinY || secWorldY > bounds.MaxY {
					continue
				}

				// Quick check: if palette is only air, skip
				if len(sec.BlockStates.Palette) == 1 && sec.BlockStates.Palette[0].Name == "minecraft:air" {
					continue
				}

				for ly := 0; ly < 16; ly++ {
					wy := secWorldY + ly
					if wy < bounds.MinY || wy > bounds.MaxY {
						continue
					}
					for lx := 0; lx < 16; lx++ {
						wx := chunkWorldX + lx
						if wx < bounds.MinX || wx > bounds.MaxX {
							continue
						}
						for lz := 0; lz < 16; lz++ {
							wz := chunkWorldZ + lz
							if wz < bounds.MinZ || wz > bounds.MaxZ {
								continue
							}

							name := getBlockFromSection(&sec, lx, ly, lz)
							if name == "minecraft:air" || name == "air" {
								continue
							}

							// Strip minecraft: prefix
							name = strings.TrimPrefix(name, "minecraft:")
							localMaterials[name]++
						}
					}
				}
			}
		}
	}

	mu.Lock()
	for k, v := range localMaterials {
		result.Materials[k] += v
	}
	mu.Unlock()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: world-scanner <world-path> [--bounds minX minY minZ maxX maxY maxZ] [--json output.json]")
		os.Exit(1)
	}

	worldPath := os.Args[1]
	regionDir := filepath.Join(worldPath, "region")

	bounds := Bounds{
		MinX: -400, MaxX: 50,
		MinY: -44, MaxY: 34,
		MinZ: -450, MaxZ: -24,
	}

	var jsonOut string

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--bounds":
			fmt.Sscan(os.Args[i+1], &bounds.MinX)
			fmt.Sscan(os.Args[i+2], &bounds.MinY)
			fmt.Sscan(os.Args[i+3], &bounds.MinZ)
			fmt.Sscan(os.Args[i+4], &bounds.MaxX)
			fmt.Sscan(os.Args[i+5], &bounds.MaxY)
			fmt.Sscan(os.Args[i+6], &bounds.MaxZ)
			i += 6
		case "--json":
			jsonOut = os.Args[i+1]
			i++
		}
	}

	fmt.Printf("Scanning %s\n", worldPath)
	fmt.Printf("Bounds: X[%d..%d] Y[%d..%d] Z[%d..%d]\n",
		bounds.MinX, bounds.MaxX, bounds.MinY, bounds.MaxY, bounds.MinZ, bounds.MaxZ)

	volume := int64(bounds.MaxX-bounds.MinX+1) * int64(bounds.MaxY-bounds.MinY+1) * int64(bounds.MaxZ-bounds.MinZ+1)
	fmt.Printf("Volume: %d blocks\n", volume)

	start := time.Now()

	files, err := filepath.Glob(filepath.Join(regionDir, "r.*.*.mca"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result := &ScanResult{
		Materials: make(map[string]int64),
		Bounds:    bounds,
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, f := range files {
		wg.Add(1)
		go scanRegionFile(f, bounds, &mu, result, &wg)
	}
	wg.Wait()

	elapsed := time.Since(start)

	// Print summary
	type matEntry struct {
		Name  string
		Count int64
	}
	var entries []matEntry
	var totalBlocks int64
	for k, v := range result.Materials {
		entries = append(entries, matEntry{k, v})
		totalBlocks += v
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })

	fmt.Printf("\nScanned in %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Total non-air blocks: %d (%.1f%% of volume)\n", totalBlocks, float64(totalBlocks)/float64(volume)*100)
	fmt.Printf("Unique materials: %d\n\n", len(entries))

	for _, e := range entries {
		pct := float64(e.Count) / float64(totalBlocks) * 100
		fmt.Printf("  %8d  %5.1f%%  %s\n", e.Count, pct, e.Name)
	}

	if jsonOut != "" {
		data, _ := json.MarshalIndent(result, "", "  ")
		os.WriteFile(jsonOut, data, 0644)
		fmt.Printf("\nJSON written to %s\n", jsonOut)
	}
}
