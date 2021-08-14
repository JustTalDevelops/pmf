package pmf

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl32"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"os"
)

// Level is the decoded level.pmf file.
type Level struct {
	// Version is the PMF version.
	Version uint8
	// Name is the name of the PMF world.
	Name string
	// Seed is the PMF world's seed.
	Seed uint32
	// Time is the time in the world.
	Time uint32
	// Spawn is the spawn position in the PMF world.
	Spawn mgl32.Vec3
	// Width is the width of the world.
	Width uint8
	// Height is the height of the world.
	Height uint8

	// chunkCache is a cache from chunk index to chunk.
	chunkCache map[int]*Chunk
	// locationMappings gets the maximum Y for a chunk location and is used for sub chunk reading.
	locationMappings map[int]uint16
	// worldPath is the path to the world.
	worldPath string
	// tiles contains a slice of all block entities in the world.
	tiles []map[string]interface{}
}

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
				"SignTextColor":               int32(-0x1000000),                                             // Colour for black text, since text dye wasn't a thing back then.
				"IgnoreLighting":              boolByte(false),                                               // Glowing text didn't exist, so we set this to false.
				"TextIgnoreLegacyBugResolved": boolByte(false),                                               // Same here.
				"Text":                        textOne + "\n" + textTwo + "\n" + textThree + "\n" + textFour, // Merge the text.
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

// Block gets a block name and properties from a position.
func (p *Level) Block(pos cube.Pos) (string, map[string]interface{}, error) {
	c, err := p.Chunk(pos.X()>>4, pos.Z()>>4)
	if err != nil {
		return "", nil, err
	}
	name, properties := c.Block(pos)

	return name, properties, nil
}

// BlockMeta gets a block's metadata at a position.
func (p *Level) BlockMeta(pos cube.Pos) (byte, error) {
	c, err := p.Chunk(pos.X()>>4, pos.Z()>>4)
	if err != nil {
		return 0, err
	}
	return c.BlockMeta(pos), nil
}

// BlockID gets a block ID at a position.
func (p *Level) BlockID(pos cube.Pos) (byte, error) {
	c, err := p.Chunk(pos.X()>>4, pos.Z()>>4)
	if err != nil {
		return 0, err
	}
	return c.BlockID(pos), nil
}

// Chunk gets a PMF chunk by it's X and Z and returns a PMFChunk.
func (p *Level) Chunk(x, z int) (*Chunk, error) {
	chunkIndex := getIndex(x, z)
	if c, ok := p.chunkCache[chunkIndex]; ok {
		return c, nil
	}

	b, err := os.ReadFile(p.worldPath + "/" + chunkFilePath(x, z))
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	result, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(result)

	info := p.locationMappings[chunkIndex]

	subChunks := make(map[uint8][]byte)
	for y := uint8(0); y < p.Height; y++ {
		t := uint16(1 << y)

		if (info & t) == t {
			subChunks[y] = buf.Next(8192)
		}
	}

	c := &Chunk{subChunks: subChunks}
	p.chunkCache[chunkIndex] = c

	return c, nil
}

// Close closes the PMF level.
func (p *Level) Close() {
	p.chunkCache = nil
	p.locationMappings = nil
}

// DecodeLevel decodes a level.pmf file from it's path and returns a Level.
func DecodeLevel(world string) (*Level, error) {
	b, err := os.ReadFile(world + "/level.pmf")
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)

	buf.Next(5) // Header.

	version, _ := buf.ReadByte()
	name := readString(buf)
	seed := readUint32(buf)
	time := readUint32(buf)
	spawnX := readFloat32(buf)
	spawnY := readFloat32(buf)
	spawnZ := readFloat32(buf)
	width, _ := buf.ReadByte()
	height, _ := buf.ReadByte()

	buf.Next(int(readUint16(buf))) // Read useless extra data.

	locationMappings := make(map[int]uint16)
	count := int(math.Pow(float64(width), 2))
	for index := 0; index < count; index++ {
		locationMappings[index] = readUint16(buf)
	}

	b, err = os.ReadFile(world + "/tiles.yml")
	if err != nil {
		return nil, err
	}

	var tiles []map[string]interface{}
	err = yaml.Unmarshal(b, &tiles)
	if err != nil {
		return nil, err
	}

	return &Level{
		Version:          version,
		Name:             name,
		Seed:             seed,
		Time:             time,
		Spawn:            mgl32.Vec3{spawnX, spawnY, spawnZ},
		Width:            width,
		Height:           height,
		worldPath:        world,
		chunkCache:       make(map[int]*Chunk),
		locationMappings: locationMappings,
		tiles:            tiles,
	}, err
}
