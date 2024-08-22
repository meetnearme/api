package indexing

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"
)

func ConvertUnixTimeToBinary(unixTime int64) string {
    // we need to start with 64 bits to account for the year, avoiding
    // trunction of it, later we shave this to remove left-padded zeroes
    //
    // IMPORTANT: left-padding is the default in golang's binary helper
    // functions, so we need to work around that scenario. This means that
    // we can't presume 32 bits is the correct starting position of truncation
    binaryStr := fmt.Sprintf("%064b", uint64(unixTime))

    // keep only characters at index 32 on (represented as 64 here)
    // this trims the left-padding to slightly less than 32 bits
    // so that we don't shave the signicant first bit, which dictates
    // the YEAR in unix timestamp
    // binaryStr = binaryStr[31:]

    return binaryStr
}

func CalculateZOrderIndex(startTime time.Time, lat, lon float64, indexType string) ([]byte, error) {
    // Get current timestamp as index creation time
    indexCreationTime := time.Now().UTC()
    startTimeUnix := startTime.Unix()
    // Convert dimensions to binary representations

    startTimeBin := ConvertUnixTimeToBinary(startTimeUnix)

    // Map floating point values to sortable unsigned integers
    // lonSortableInt := mapFloatToSortableInt32(lon)
    // latSortableInt := mapFloatToSortableInt32(lat)

    lonSortableInt := mapFloatToSortableInt64(float64(lon))
    latSortableInt := mapFloatToSortableInt64(float64(lat))


    // Convert sortable integers to binary string
    lonBin := fmt.Sprintf("%032b", lonSortableInt)
    latBin := fmt.Sprintf("%032b", latSortableInt)

    // Interleave binary representations
    var zIndexBin string

    for i := 0; i < 32; i++ {
        zIndexBin += startTimeBin[i : i+1]

            zIndexBin += lonBin[i : i+1]
            zIndexBin += latBin[i : i+1]
    }

    // Convert binary string to byte slice
    zIndexBytes := make([]byte, len(zIndexBin)/8)
    for i := 0; i < len(zIndexBin); i += 8 {
        b, _ := strconv.ParseUint(zIndexBin[i:i+8], 2, 8)
        zIndexBytes[i/8] = byte(b)
    }

    if indexType == "min" {
        zIndexBytes = append(zIndexBytes, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xFF}...)
    } else if indexType == "max" {
        zIndexBytes = append(zIndexBytes, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}...)
    } else {
        // Append index creation time as bytes
        indexCreationTimeBytes, _ := indexCreationTime.MarshalBinary()
        zIndexBytes = append(zIndexBytes, indexCreationTimeBytes...)
    }

    return zIndexBytes, nil
}

func mapFloatToSortableInt32(floatValue float32) uint32 {
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

func mapFloatToSortableInt64(floatValue float64) uint64 {
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
        // XOR to flip all bits, makes negative float come first as int
        sortableBytes = make([]byte, 8)
        for i, b := range floatBytes {
            sortableBytes[i] = b ^ 0xFF
        }
    }
    // Convert mapped bytes to an unsigned integer
    sortableInt := binary.BigEndian.Uint64(sortableBytes)

    return sortableInt
}
