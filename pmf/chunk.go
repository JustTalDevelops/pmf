package pmf

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
)

// Chunk is a PMF style chunk.
type Chunk struct {
	// subChunks is a map of Y level to sub chunk.
	subChunks map[uint8][]byte
}

// NewEmptyChunk creates a new empty chunk.
func NewEmptyChunk() *Chunk {
	return &Chunk{
		subChunks: make(map[uint8][]byte),
	}
}

// Block gets a block name and properties from a position.
func (c *Chunk) Block(pos cube.Pos) (string, map[string]interface{}, error) {
	id, err := c.BlockID(pos)
	if err != nil {
		return "", nil, err
	}
	metadata, err := c.BlockMeta(pos)
	if err != nil {
		return "", nil, err
	}
	converted := conversion[oldBlock{id: id, metadata: metadata}]
	return converted.name, converted.properties, nil
}

// SetBlockID sets the block ID at a position.
func (c *Chunk) SetBlockID(pos cube.Pos, id byte) error {
	if !validatePos(pos) {
		return fmt.Errorf("block pos not valid")
	}

	chunkY := pos.Y() >> 4
	if chunkY >= len(c.subChunks) {
		return fmt.Errorf("chunk pos not valid")
	}
	c.subChunks[uint8(chunkY)][idIndex(pos)] = id
	return nil
}

// BlockID gets a block ID at a position.
func (c *Chunk) BlockID(pos cube.Pos) (byte, error) {
	if !validatePos(pos) {
		return 0, fmt.Errorf("block pos not valid")
	}

	chunkY := pos.Y() >> 4
	if chunkY >= len(c.subChunks) {
		return 0, fmt.Errorf("chunk pos not valid")
	}
	return c.subChunks[uint8(chunkY)][idIndex(pos)], nil
}

// SetBlockMeta sets the block metadata at a position.
func (c *Chunk) SetBlockMeta(pos cube.Pos, meta byte) error {
	if !validatePos(pos) {
		return fmt.Errorf("block pos not valid")
	}

	chunkY := pos.Y() >> 4
	if chunkY >= len(c.subChunks) {
		return fmt.Errorf("chunk pos not valid")
	}

	meta &= 0x0F
	metaInd := metaIndex(pos)
	oldMeta := c.subChunks[uint8(chunkY)][metaInd]
	if (pos.Y() & 1) == 0 {
		meta = (oldMeta & 0xF0) | meta
	} else {
		meta = (meta << 4) | (oldMeta & 0x0F)
	}

	c.subChunks[uint8(chunkY)][metaInd] = meta
	return nil
}

// BlockMeta gets the metadata at a block at a position.
func (c *Chunk) BlockMeta(pos cube.Pos) (byte, error) {
	if !validatePos(pos) {
		return 0, fmt.Errorf("block pos not valid")
	}

	chunkY := pos.Y() >> 4
	if chunkY >= len(c.subChunks) {
		return 0, fmt.Errorf("chunk pos not valid")
	}

	meta := c.subChunks[uint8(chunkY)][metaIndex(pos)]
	if (pos.Y() & 1) == 0 {
		meta = meta & 0x0F
	} else {
		meta = meta >> 4
	}
	return meta, nil
}

// metaIndex gets the index of the metadata at a position.
func metaIndex(pos cube.Pos) int {
	aX, aZ, aY := offset(pos)
	return (aY >> 1) + 16 + (aX << 5) + (aZ << 9)
}

// idIndex returns the index of the block ID in the sub chunk.
func idIndex(pos cube.Pos) int {
	aX, aZ, aY := offset(pos)
	return aY + (aX << 5) + (aZ << 9)
}

// offset returns the offsets of the block pos.
func offset(pos cube.Pos) (int, int, int) {
	chunkX := pos.X() >> 4
	chunkY := pos.Y() >> 4
	chunkZ := pos.Z() >> 4

	aX := pos.X() - (chunkX << 4)
	aZ := pos.Z() - (chunkZ << 4)
	aY := pos.Y() - (chunkY << 4)
	return aX, aZ, aY
}

// validatePos checks if a position is valid.
func validatePos(pos cube.Pos) bool {
	return pos.Y() > 127 || pos.Y() < 0 || pos.X() < 0 || pos.Z() < 0 || pos.X() > 255 || pos.Z() > 255
}
