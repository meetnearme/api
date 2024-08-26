package indexing

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// IMPORTANT:
// ================================
// Tather right-padding with zeros to the max 64 bit length, we're
// left-padding with zeros to accommodate up to 35 integer precision
// which is roughly the year 3,000. If we used 32 bits, we would run
// into "The 2038 problem" around 14 years from the time of this writing
// if we used the full 64 bits, our index precision would be diluted by
// the large span of 292 billion years that 64 bit unix time can represent

func ConvertUnixTimeToBinary(unixTime int64) string {

    // Create a 35-character base of zeroes
    base := strings.Repeat("0", 35)

    // Convert to unpadded binary 64-bit representation
    binaryStr := fmt.Sprintf("%b", unixTime)

    // Right-align the binaryStr within the 35-character precision limit
    // (year 3,000) and right-pad with 29 zeroes to ensure 64-bit length
    alignedStr := base[:35-len(binaryStr)] + binaryStr

    result := alignedStr + strings.Repeat("0", 29)

    return result
}

func BinToDecimal(binaryStr string) (*big.Int, error) {
    decimal := new(big.Int)
    _, ok := decimal.SetString(binaryStr, 2)
    if !ok {
        return nil, fmt.Errorf("failed to convert binary string to decimal")
    }
    return decimal, nil
}

func CalculateZOrderIndex(tm time.Time, lat, lon float64, indexType string) (string, error) {
    // Get current timestamp as index creation time
    indexCreationTime := time.Now().UTC()
    tmUnix := tm.Unix()

    // Convert dimensions to binary representations
    tmBin := ConvertUnixTimeToBinary(tmUnix)
    log.Println("tmBin: ", tmBin)

    log.Println("lon: ", lon)
    log.Println("lat: ", lat)

    lonSortableBinStr := mapFloatToSortableBinaryString(lon)
    latSortableBinStr := mapFloatToSortableBinaryString(lat)

    log.Println("lonSortableBinStr: ", lonSortableBinStr)
    decimal, _ := BinToDecimal(lonSortableBinStr)
    log.Println("decimal: ", decimal)
    log.Println("latSortableBinStr: ", latSortableBinStr)
    decimal, _ = BinToDecimal(latSortableBinStr)
    log.Println("decimal: ", decimal)

    log.Println(">>> LINE 74 <<< tmBin: ", tmBin)


    // latSortableBinStr = "1111111111111111111111111111111111111111111111111111111111111111"

    // lonSortableBinStr = "0000000000000000000000000000000000000000000000000000000000000000"

    // tmBin = "0000000000000000000000000000000000000000000000000000000000000000"

    // Interleave binary representations
    var zIndexBin string

    for i := 0; i < 64; i++ {
        zIndexBin += tmBin[i : i+1]
        zIndexBin += lonSortableBinStr[i : i+1]
        zIndexBin += latSortableBinStr[i : i+1]
    }

    if indexType == "min" {
        zIndexBin += "0000000000000000000000000000000000000000000000000000000000000001"
    } else if indexType == "max" {
        zIndexBin += "1111111111111111111111111111111111111111111111111111111111111111"
    } else {
        binaryStr := strconv.FormatInt(indexCreationTime.Unix(), 2)

        // Left-pad with zeros to ensure 64-bit length
        zIndexBin += strings.Repeat("0", 64-len(binaryStr)) + binaryStr
    }

    return zIndexBin, nil
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

func mapFloatToSortableBinaryString(floatValue float64) string {
    // Convert float64 to bits
    floatValue += 2000;
    bits := math.Float64bits(floatValue)

    log.Println("\n\n\n>>>> floatValue:", floatValue)
    log.Println(">>>> bits:", bits)
    // Convert bits to binary string
    binaryStr := fmt.Sprintf("%064b", bits)

    if floatValue < 0 {
        log.Println("ERR: negative integers are not supported")
        return ""
    }

    // For positive values, just flip the sign bit
    retVal := "1" + binaryStr[1:]
    log.Println(">>>> retVal:", retVal)
    return retVal
}

func mapFloatToSortableInt64(floatValue float64) uint64 {
    // Convert float64 to byte slice
    floatBytes := make([]byte, 8)
    binary.LittleEndian.PutUint64(floatBytes, math.Float64bits(floatValue))

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
    sortableInt := binary.LittleEndian.Uint64(sortableBytes)

    return sortableInt
}
