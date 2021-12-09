package pmf

import (
	"bytes"
	"compress/gzip"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl32"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"os"
	"path"
)

// currentVersion is the current version of the PMF format.
const currentVersion = 0x00

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

// Block gets a block name and properties from a position.
func (p *Level) Block(pos cube.Pos) (string, map[string]interface{}, error) {
	c, err := p.Chunk(pos.X()>>4, pos.Z()>>4)
	if err != nil {
		return "", nil, err
	}
	return c.Block(pos)
}

// BlockMeta gets a block's metadata at a position.
func (p *Level) BlockMeta(pos cube.Pos) (byte, error) {
	c, err := p.Chunk(pos.X()>>4, pos.Z()>>4)
	if err != nil {
		return 0, err
	}
	return c.BlockMeta(pos)
}

// BlockID gets a block ID at a position.
func (p *Level) BlockID(pos cube.Pos) (byte, error) {
	c, err := p.Chunk(pos.X()>>4, pos.Z()>>4)
	if err != nil {
		return 0, err
	}
	return c.BlockID(pos)
}

// Chunk gets a PMF chunk by its X and Z and returns a PMFChunk.
func (p *Level) Chunk(x, z int) (*Chunk, error) {
	chunkIndex := getIndex(x, z)
	if c, ok := p.chunkCache[chunkIndex]; ok {
		return c, nil
	}

	b, err := os.ReadFile(path.Join(p.worldPath, chunkFilePath(x, z)))
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

// NewLevel creates a new PMF level from a path.
func NewLevel(folderPath, levelName string, seed uint32, width, height byte, spawn mgl32.Vec3) (*Level, error) {
	f, err := os.Create(path.Join(folderPath, "level.pmf"))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	buf.Write(make([]byte, 5))    // Header.
	buf.WriteByte(currentVersion) // Version (0x00).

	writeString(buf, levelName)
	writeUint32(buf, seed)
	writeUint32(buf, 0) // Time.

	writeFloat32(buf, spawn.X())
	writeFloat32(buf, spawn.Y())
	writeFloat32(buf, spawn.Z())

	buf.WriteByte(width)  // Width.
	buf.WriteByte(height) // Height.

	writeUint16(buf, 0) // Extra data length.

	count := int(math.Pow(float64(width), 2))
	locationMappings := make(map[int]uint16, count)
	for index := 0; index < count; index++ {
		writeUint16(buf, 0) // Location mapping.
	}

	_, err = f.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}

	return &Level{
		Version:          currentVersion,
		Name:             levelName,
		Seed:             seed,
		Spawn:            spawn,
		Width:            width,
		Height:           height,
		chunkCache:       make(map[int]*Chunk),
		locationMappings: locationMappings,
		worldPath:        folderPath,
	}, nil
}

// DecodeLevel decodes a level.pmf file from its path and returns a Level.
func DecodeLevel(folderPath string) (*Level, error) {
	b, err := os.ReadFile(path.Join(folderPath, "level.pmf"))
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

	b, err = os.ReadFile(path.Join(folderPath, "tiles.yml"))
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
		Width:            width,
		Height:           height,
		worldPath:        folderPath,
		locationMappings: locationMappings,
		tiles:            tiles,
		Spawn:            mgl32.Vec3{spawnX, spawnY, spawnZ},
		chunkCache:       make(map[int]*Chunk),
	}, err
}
