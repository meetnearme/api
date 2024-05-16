package indexing

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"
)

func CalculateZOrderIndex(startTime time.Time, lat, lon float32, indexType string) ([]byte, error) {
    // Get current timestamp as index creation time
    indexCreationTime := time.Now().UTC()

    // Convert dimensions to binary representations
    startTimeUnix := startTime.Unix()
    startTimeBin := strconv.FormatInt(startTimeUnix, 2)
    startTimeBin = fmt.Sprintf("%032s", startTimeBin)

    // Map floating point values to sortable unsigned integers
    lonSortableInt := mapFloatToSortableInt(lon)
    latSortableInt := mapFloatToSortableInt(lat)

    // Convert sortable integers to binary string
    lonBin := strconv.FormatUint(uint64(lonSortableInt), 2)
    lonBin = fmt.Sprintf("%032s", lonBin)
    latBin := strconv.FormatUint(uint64(latSortableInt), 2)
    latBin = fmt.Sprintf("%032s", latBin)

    // Interleave binary representations
    var zIndexBin string
    for i := 0; i < 32; i++ {
        zIndexBin += startTimeBin[i : i+1]
        if i < 32 {
            zIndexBin += lonBin[i : i+1]
            zIndexBin += latBin[i : i+1]
        }
    }

    // Convert binary string to byte slice
    zIndexBytes := make([]byte, len(zIndexBin)/8)
    for i := 0; i < len(zIndexBin); i += 8 {
        b, _ := strconv.ParseUint(zIndexBin[i:i+8], 2, 8)
        zIndexBytes[i/8] = byte(b)
    }

    // Append index creation time as bytes
    indexCreationTimeBytes, _ := indexCreationTime.MarshalBinary()
    zIndexBytes = append(zIndexBytes, indexCreationTimeBytes...)

    if indexType == "min" {
        zIndexBytes = append(zIndexBytes, make([]byte, 8)...)
    } else if indexType == "max" {
        zIndexBytes = append(zIndexBytes, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}...)
    }

    return zIndexBytes, nil
}

func mapFloatToSortableInt(floatValue float32) uint32 {
    // Convert float64 to byte slice
    floatBytes := make([]byte, 8)
    binary.BigEndian.PutUint32(floatBytes, math.Float32bits(floatValue))

    var sortableBytes []byte

    if floatValue >= 0 {
        // XOR op for flipping first bit to make unsigned int representation of positive float after negative
        sortableBytes = make([]byte, 8)
        sortableBytes[0] = floatBytes[0] ^ 0x80
        copy(sortableBytes[1:], floatBytes[1:])
    } else {
        // XOR to flit all bits, makes negative float come first as int
        sortableBytes = make([]byte, 8)
        for i, b := range floatBytes {
            sortableBytes[i] = b ^ 0xFF
        }

    } 
    // Convert mapped bytes to an unsigned integer
    sortableInt :=  binary.BigEndian.Uint32(sortableBytes)

    return sortableInt
} 
