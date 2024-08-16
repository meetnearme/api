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
    lonBin := fmt.Sprintf("%032b", lonSortableInt)
    latBin := fmt.Sprintf("%032b", latSortableInt)

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

func DeriveValuesFromZOrder(zOrderIndex []byte) (startTime time.Time, lat, lon float32, err error) {
    if len(zOrderIndex) < 12 {
        return time.Time{}, 0, 0, fmt.Errorf("invalid z-order index length")
    }

    // Extract the interleaved binary string
    zIndexBin := ""
    for i := 0; i < 12; i++ {
        zIndexBin += fmt.Sprintf("%08b", zOrderIndex[i])
    }

    // De-interleave the binary string
    startTimeBin := ""
    lonBin := ""
    latBin := ""
    for i := 0; i < 96; i += 3 {
        startTimeBin += zIndexBin[i : i+1]
        lonBin += zIndexBin[i+1 : i+2]
        latBin += zIndexBin[i+2 : i+3]
    }

    // Convert binary strings to values
    startTimeUnix, err := strconv.ParseInt(startTimeBin, 2, 64)
    if err != nil {
        return time.Time{}, 0, 0, fmt.Errorf("error parsing start time: %v", err)
    }
    startTime = time.Unix(startTimeUnix, 0)

    lonSortableInt, err := strconv.ParseUint(lonBin, 2, 32)
    if err != nil {
        return time.Time{}, 0, 0, fmt.Errorf("error parsing longitude: %v", err)
    }

    latSortableInt, err := strconv.ParseUint(latBin, 2, 32)
    if err != nil {
        return time.Time{}, 0, 0, fmt.Errorf("error parsing latitude: %v", err)
    }

    lon = mapSortableIntToFloat(uint32(lonSortableInt))
    lat = mapSortableIntToFloat(uint32(latSortableInt))

    return startTime, lat, lon, nil
}

func mapSortableIntToFloat(sortableInt uint32) float32 {
    floatBytes := make([]byte, 4)
    binary.BigEndian.PutUint32(floatBytes, sortableInt)

    if sortableInt&0x80000000 != 0 {
        // Positive number, flip the first bit back
        floatBytes[0] ^= 0x80
    } else {
        // Negative number, flip all bits back
        for i := range floatBytes {
            floatBytes[i] ^= 0xFF
        }
    }

    return math.Float32frombits(binary.BigEndian.Uint32(floatBytes))
}
