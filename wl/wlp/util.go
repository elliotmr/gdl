package wlp

import (
	"math"
	"unsafe"
	"encoding/binary"
)

var hostByteOrder binary.ByteOrder

func init() {
	var endianCheck uint32 = 0x1
	b := (*[4]byte)(unsafe.Pointer(&endianCheck))
	if b[0] == 1 {
		hostByteOrder = binary.LittleEndian
	} else {
		hostByteOrder = binary.BigEndian
	}
}

func fixedToFloat64(fixed int32) float64 {
	i := ((1023 + 44) << 52) + (1 << 51) + uint64(fixed);
	return math.Float64frombits(i) - (3 << 43)
}

func float64ToFixed(float float64) int32 {
	float += 3 << 43
	return int32(math.Float64bits(float))
}

func fixedToInt(fixed int32) int {
	return int(*(*int32)(unsafe.Pointer(&fixed))) / 256
}

func intToFixed(i int) int32 {
	i32 := int32(i * 256)
	return *(*int32)(unsafe.Pointer(&i32))
}

func DecodeHeader(buf []byte) (id uint32, opcode uint16, size int) {
	id = hostByteOrder.Uint32(buf[:4])
	arg2 := hostByteOrder.Uint32(buf[4:8])
	opcode = uint16(arg2 & 0xFF)
	size = int(arg2 >> 16)
	return
}