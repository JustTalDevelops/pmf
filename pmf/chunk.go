package pmf

import "github.com/df-mc/dragonfly/server/block/cube"

// PMFChunk is a PMF style chunk.
type PMFChunk struct {
	// subChunks is a map of Y level to sub chunk.
	subChunks map[uint8][]byte
}

// Block gets a block name and properties from a position.
func (p *PMFChunk) Block(pos cube.Pos) (string, map[string]interface{}) {
	id := p.BlockID(pos)
	meta := p.BlockMeta(pos)

	converted := conversion[oldBlock{id: id, metadata: meta}]

	return converted.name, converted.properties
}

// BlockID gets a block ID at a position.
func (p *PMFChunk) BlockID(pos cube.Pos) byte {
	if pos.Y() > 127 || pos.Y() < 0 || pos.X() < 0 || pos.Z() < 0 || pos.X() > 255 || pos.Z() > 255 {
		return 0
	}

	chunkY := pos.Y() >> 4
	if chunkY >= len(p.subChunks) {
		return 0
	}

	chunkX := pos.X() >> 4
	chunkZ := pos.Z() >> 4

	aX := pos.X() - (chunkX << 4)
	aZ := pos.Z() - (chunkZ << 4)
	aY := pos.Y() - (chunkY << 4)

	index := aY + (aX << 5) + (aZ << 9)

	return p.subChunks[uint8(chunkY)][index]
}

// BlockMeta gets the metadata at a block at a position.
func (p *PMFChunk) BlockMeta(pos cube.Pos) byte {
	if pos.Y() > 127 || pos.Y() < 0 || pos.X() < 0 || pos.Z() < 0 || pos.X() > 255 || pos.Z() > 255 {
		return 0
	}

	chunkY := pos.Y() >> 4
	if chunkY >= len(p.subChunks) {
		return 0
	}

	chunkX := pos.X() >> 4
	chunkZ := pos.Z() >> 4

	aX := pos.X() - (chunkX << 4)
	aZ := pos.Z() - (chunkZ << 4)
	aY := pos.Y() - (chunkY << 4)

	index := (aY >> 1) + 16 + (aX << 5) + (aZ << 9)

	m := p.subChunks[uint8(chunkY)][index]
	if (pos.Y() & 1) == 0 {
		m = m & 0x0F
	} else {
		m = m >> 4
	}

	return m
}