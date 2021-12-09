package pmf

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
)

// Convert converts the PMF level to a provider.
func (p *Level) Convert(prov *mcdb.Provider) error {
	settings := prov.Settings()
	settings.Name = p.Name
	settings.Spawn = cube.Pos{int(p.Spawn.X()), int(p.Spawn.Y()), int(p.Spawn.Z())}
	settings.Time = int64(p.Time)
	prov.SaveSettings(settings)

	airRuntimeID, ok := chunk.StateToRuntimeID("minecraft:air", nil)
	if !ok {
		panic("could not find air runtime id")
	}

	chunks := make(map[world.ChunkPos]*chunk.Chunk)
	for x := 0; x < 256; x++ {
		for z := 0; z < 256; z++ {
			for y := 0; y < 128; y++ {
				name, properties, err := p.Block(cube.Pos{x, y, z})
				if err != nil {
					panic(err)
				}
				if name == "minecraft:air" {
					continue
				}

				chunkPos := world.ChunkPos{int32(x >> 4), int32(z >> 4)}
				ch, ok := chunks[chunkPos]
				if !ok {
					ch = chunk.New(airRuntimeID)
					for x := uint8(0); x < 17; x++ {
						for z := uint8(0); z < 17; z++ {
							ch.SetBiomeID(x, z, 1) // The only biome in PM when PMF was a thing was plains.
						}
					}

					chunks[chunkPos] = ch
				}

				rid, ok := chunk.StateToRuntimeID(name, properties)
				if !ok {
					panic(fmt.Errorf("could not find runtime id for state: %v, %v", name, properties))
				}

				ch.SetRuntimeID(uint8(x), int16(y), uint8(z), 0, rid)
			}
		}
	}

	blockEntities := make(map[world.ChunkPos][]map[string]interface{})
	for _, t := range p.tiles {
		tileType := t["id"]

		switch tileType {
		case "Sign":
			x, y, z := t["x"].(int), t["y"].(int), t["z"].(int)

			chunkPos := world.ChunkPos{int32(x >> 4), int32(z >> 4)}
			if _, ok := blockEntities[chunkPos]; !ok {
				blockEntities[chunkPos] = []map[string]interface{}{}
			}

			textOne, textTwo, textThree, textFour := t["Text1"].(string), t["Text2"].(string), t["Text3"].(string), t["Text4"].(string)

			data := map[string]interface{}{
				"id":                          "Sign",
				"SignTextColor":               int32(-0x1000000),
				"IgnoreLighting":              boolByte(false),
				"TextIgnoreLegacyBugResolved": boolByte(false),
				"Text":                        textOne + "\n" + textTwo + "\n" + textThree + "\n" + textFour,
			}
			data["x"], data["y"], data["z"] = int32(x), int32(y), int32(z)

			blockEntities[chunkPos] = append(blockEntities[chunkPos], data)
		}
	}

	for pos, ch := range chunks {
		err := prov.SaveChunk(pos, ch)
		if err != nil {
			return err
		}
		err = prov.SaveBlockNBT(pos, blockEntities[pos])
		if err != nil {
			return err
		}
	}

	return nil
}
