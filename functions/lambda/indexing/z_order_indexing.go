package indexing

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"
)

func CalculateZOrderIndex(startTime time.Time, lat, lon float64, indexType string) []byte {
    // Get current timestamp as index creation time
    indexCreationTime := time.Now().UTC()

    // convert dimensions to binary representations
    startTimeBin := strconv.FormatInt(startTime.Unix(), 2)
    startTimeBin = fmt.Sprintf("%032x", startTimeBin)

    // Map floating point values to sortable unsigned integers
    lonSortableInt := mapFloatToSortableInt(lon)
    latSortableInt := mapFloatToSortableInt(lat)

    // Convert sortable integers to binary string
    lonBin := strconv.FormatUint(lonSortableInt, 2)
    lonBin = fmt.Sprintf("%032s", lonBin)
    latBin := strconv.FormatUint(latSortableInt, 2)
    latBin = fmt.Sprintf("%032s", latBin)

    // Interleave binary representations
    var zIndexBin string
    for i := 0; i < 32; i++ {
        zIndexBin += startTimeBin[i: i+1]
        zIndexBin += lonBin[i: i+1]
        zIndexBin += latBin[i: i+1]
    } 

    zIndexWithCreationTimeBin := zIndexBin + string(indexCreationTime)

    // convert binary string to byte slice
    zIndexBytes, _ := hex.Decode([]byte(zIndexWithCreationTimeBin))

    return zIndexBytes
}

func mapFloatToSortableInte(floatValue float64) int {
    // Convert float64 to byte slice
    floatBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(floatBytes, math.Float64bits(floatValue))

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

        // Convert mapped bytes to an unsigned integer
        sortableInt :=  binary.BigEndian.Uint64(sortableBytes)

        return int(sortableInt)
    } 
} 
