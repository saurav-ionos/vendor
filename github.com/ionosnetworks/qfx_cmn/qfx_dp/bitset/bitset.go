package bitset

import (
	"math"
)

func CreateBitVector(length uint32) (vec []uint64) {
	numElem := length / 64
	if length%64 != 0 {
		numElem += 1
	}
	vec = make([]uint64, numElem)
	length = numElem
	return
}

func TestBit(vec []uint64, bitn uint32) (result bool) {
	bitn -= 1
	index := bitn >> 6
	if index < uint32(len(vec)) {
		offset := bitn & 63
		element := vec[index]
		if element&(1<<offset) == 0 {
			result = false
		} else {
			result = true
		}
	} else {
		result = false
	}
	return result
}

func SetBit(vec []uint64, bitn uint32) {
	bitn -= 1
	index := bitn >> 6
	if index < uint32(len(vec)) {
		offset := bitn & 63
		vec[index] |= (0x1 << offset)
	}
}

func ClearBit(vec []uint64, bitn uint32) {
	bitn -= 1
	index := bitn >> 6
	if index < uint32(len(vec)) {
		offset := bitn & 63
		vec[index] &^= (0x1 << offset)
	}
}

func Ffs(vec []uint64) uint32 {
	var index uint32 = 0
	for index = 0; index < uint32(len(vec)) &&
		vec[index] == math.MaxUint64; index++ {
	}

	if index == uint32(len(vec)) {
		return 64 * index
	}

	element := vec[index]
	var offset uint32 = 0
	for offset = 0; offset < 64; offset++ {
		if element&0x1 == 0 {
			break
		}
		element >>= 1
	}
	return ((64 * index) + offset + 1)
}

func Get64Chunks(vec []uint64, start uint32) (mask uint64) {
	length := uint32(len(vec))
	if start == 0 || length == 0 {
		mask = 0xFFFFFFFFFFFFFFFF
		return
	}
	bitn := start - 1
	index := bitn >> 6
	offset := bitn & 63
	mask = 0
	if offset == 0 {
		mask = mask | vec[index]
	} else if (length - 1) == index {
		mask = mask | (vec[index] >> (offset))
	} else {
		mask = mask | (vec[index] >> (offset)) | (vec[index+1] << (64 - (offset)))
	}
	return
}

// Returns the leftmost bit(MSB) set in the chunkArray
func Fls(vec []uint64) uint32 {
	last_index := uint32(len(vec)) - 1
	element := vec[last_index]
	lbit := math.Log2(float64(element)) + 1
	return 64*last_index + uint32(lbit)
}

// Test if given "bit" is set in the given "num"
func TestNthBit(num uint64, bit uint32) bool {
	var index uint64 = 1
	if num&(index<<bit) == 1 {
		return true
	}
	return false
}

// Masks out extra bits - bits representing chunk numbers > total number of chunks
func MaskOut(bmask uint64, total uint32, start uint32) uint64 {
	if start == 0 || total == 0 {
		return bmask
	}
	diff := total - start
	if diff < 63 {
		offset := (start + 63) - total
		mask := uint64(0xFFFFFFFFFFFFFFFF >> offset)
		mask = mask & bmask
		return mask
	}
	return bmask
}
