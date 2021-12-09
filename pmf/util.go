package pmf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// chunkFilePath gets a chunk's file path from it's X and Z.
func chunkFilePath(x int, z int) string {
	return fmt.Sprintf("chunks/%v.%v.pmc", z, x)
}

// getIndex gets a chunk index from an X and Z.
func getIndex(x, z int) int {
	return (z << 4) + x
}

// boolByte returns 1 if the bool passed is true, or 0 if it is false.
func boolByte(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// writeString writes a string to a buffer.
func writeString(buf *bytes.Buffer, v string) {
	writeUint16(buf, uint16(len(v)))
	buf.Write([]byte(v))
}

// readString reads a string from a buffer.
func readString(buf *bytes.Buffer) string {
	return string(buf.Next(int(readUint16(buf))))
}

// writeFloat32 writes a float32 to a buffer.
func writeFloat32(buf *bytes.Buffer, v float32) {
	writeUint32(buf, math.Float32bits(v))
}

// readFloat32 reads a float32 from a buffer.
func readFloat32(buf *bytes.Buffer) float32 {
	return math.Float32frombits(readUint32(buf))
}

// writeUint32 writes an uint32 to a buffer.
func writeUint32(buf *bytes.Buffer, v uint32) {
	_ = binary.Write(buf, binary.BigEndian, v)
}

// readUint32 reads an uint32 from a buffer.
func readUint32(buf *bytes.Buffer) uint32 {
	return binary.BigEndian.Uint32(buf.Next(4))
}

// writeUint16 writes an uint16 to a buffer.
func writeUint16(buf *bytes.Buffer, v uint16) {
	_ = binary.Write(buf, binary.BigEndian, v)
}

// readUint16 reads an uint16 from a buffer.
func readUint16(buf *bytes.Buffer) uint16 {
	return binary.BigEndian.Uint16(buf.Next(2))
}
