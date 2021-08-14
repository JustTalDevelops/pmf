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
	"io/ioutil"
	"math"
	"os"
)

// PMFLevel is the decoded level.pmf file.
type PMFLevel struct {
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
	chunkCache map[int]*PMFChunk
	// locationMappings gets the maximum Y for a chunk location and is used for sub chunk reading.
	locationMappings map[int]uint16
	// worldPath is the path to the world.
	worldPath string
}

// Convert converts the PMF level to a provider.
func (p *PMFLevel) Convert(prov *mcdb.Provider) error {
	settings := prov.Settings()
	settings.Name = p.Name
	settings.Spawn = cube.Pos{int(p.Spawn.X()), int(p.Spawn.Y()), int(p.Spawn.Z())}
	settings.Time = int64(p.Time)

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

	for pos, ch := range chunks {
		err := prov.SaveChunk(pos, ch)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Block gets a block name and properties from a position.
func (p *PMFLevel) Block(pos cube.Pos) (string, map[string]interface{}, error) {
	c, err := p.Chunk(pos.X() >> 4, pos.Z() >> 4)
	if err != nil {
		return "", nil, err
	}
	name, properties := c.Block(pos)

	return name, properties, nil
}

// BlockMeta gets a block's metadata at a position.
func (p *PMFLevel) BlockMeta(pos cube.Pos) (byte, error) {
	c, err := p.Chunk(pos.X() >> 4, pos.Z() >> 4)
	if err != nil {
		return 0, err
	}
	return c.BlockMeta(pos), nil
}

// BlockID gets a block ID at a position.
func (p *PMFLevel) BlockID(pos cube.Pos) (byte, error) {
	c, err := p.Chunk(pos.X() >> 4, pos.Z() >> 4)
	if err != nil {
		return 0, err
	}
	return c.BlockID(pos), nil
}

// Chunk gets a PMF chunk by it's X and Z and returns a PMFChunk.
func (p *PMFLevel) Chunk(x, z int) (*PMFChunk, error) {
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

	chunk := &PMFChunk{subChunks: subChunks}
	p.chunkCache[chunkIndex] = chunk

	return chunk, nil
}

// Close closes the PMF level.
func (p *PMFLevel) Close() {
	p.chunkCache = nil
	p.locationMappings = nil
}

// DecodePMF decodes a level.pmf file from it's path and returns a PMFLevel.
func DecodePMF(world string) (*PMFLevel, error) {
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

	return &PMFLevel{
		Version: version,
		Name: name,
		Seed: seed,
		Time: time,
		Spawn: mgl32.Vec3{spawnX, spawnY, spawnZ},
		Width: width,
		Height: height,
		worldPath: world,
		chunkCache: make(map[int]*PMFChunk),
		locationMappings: locationMappings,
	}, err
}