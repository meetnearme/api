package indexing

import (
	"encoding/binary"
	// "encoding/hex"
	"fmt"
	"math"
	"strconv"
	// "strings"
	"time"
)

func CalculateZOrderIndex(startTime time.Time, lat, lon float64, indexType string) ([]byte, error) {
    // Get current timestamp as index creation time
    indexCreationTime := time.Now().UTC()

    // Convert dimensions to binary representations
    startTimeUnix := startTime.Unix()
    startTimeBin := strconv.FormatInt(startTimeUnix, 2)
    startTimeBin = fmt.Sprintf("%064s", startTimeBin)

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
    for i := 0; i < 64; i++ {
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
    // // Get current timestamp as index creation time
    // indexCreationTime := time.Now().UTC()

    // // convert dimensions to binary representations
    // startTimeBin := strconv.FormatInt(startTime.Unix(), 2)
    // startTimeBin = fmt.Sprintf("%064s", startTimeBin)

    // // Map floating point values to sortable unsigned integers
    // lonSortableInt := mapFloatToSortableInt(lon)
    // latSortableInt := mapFloatToSortableInt(lat)

    // // Convert sortable integers to binary string
    // lonBin := strconv.FormatUint(lonSortableInt, 2)
    // lonBin = fmt.Sprintf("%032s", lonBin)
    // latBin := strconv.FormatUint(latSortableInt, 2)
    // latBin = fmt.Sprintf("%032s", latBin)

    // // Interleave binary representations
    // var zIndexBin string
    // for i := 0; i < 64; i++ {
    //     zIndexBin += startTimeBin[i: i+1]
    //     if i < 32 {
    //         zIndexBin += lonBin[i: i+1]
    //         zIndexBin += latBin[i: i+1]
    //     }
    // } 

    // zIndexWithCreationTimeBin := zIndexBin + indexCreationTime.Format(time.RFC3339)

    // if indexType == "min" {
    //     zIndexBin = zIndexBin + strings.Repeat("0", 64)
    // } else if indexType == "max" {
    //     zIndexBin = zIndexBin + strings.Repeat("1", 64)
    // } else {
    //     zIndexBin = zIndexWithCreationTimeBin
    // } 


    // // convert binary string to byte slice
    // zIndexBytes := make([]byte, hex.DecodedLen(len(zIndexBin)))
    // _, err := hex.Decode(zIndexBytes, []byte(zIndexBin))
    // if err != nil {
    //     return nil, fmt.Errorf("Error in the decoding of Z Index with creation time %v", err)
    // }

    // return zIndexBytes, nil
}

func mapFloatToSortableInt(floatValue float64) uint64 {
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

    } 
    // Convert mapped bytes to an unsigned integer
    sortableInt :=  binary.BigEndian.Uint64(sortableBytes)

    return sortableInt
} 
