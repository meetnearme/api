package indexing

import (
	"encoding/base64"
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
// Rather right-padding with zeros to the max 64 bit length, we're
// left-padding with zeros to accommodate up to 35 integer precision
// which is roughly the year 3,000. If we used 32 bits, we would run
// into "The 2038 problem" around 14 years from the time of this writing
// if we used the full 64 bits, our index precision would be diluted by
// the large span of 292 billion years that 64 bit unix time can represent

func ConvertUnixTimeToShiftedBinary(unixTime int64) string {
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
    tmBin := ConvertUnixTimeToShiftedBinary(tmUnix)
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

func DecodeZOrderIndex(zOrderIndex string) (tm time.Time, lat, lon float64, trailingBits string) {
    // Convert the base64 input string to a byte array
    byteArray, err := base64.StdEncoding.DecodeString(zOrderIndex)
    if err != nil {
        log.Printf("Error decoding base64 string: %v", err)
        return
    }

    log.Printf("Decoded byte array: %v", byteArray)

    binaryString := ""
    for _, b := range byteArray {
        binaryString += fmt.Sprintf("%08b", b)
    }

    log.Printf("Binary string: %s", binaryString)

    // "Unzip" the first 192 characters, 64 each for tm, lon, lat
    tmBin := ""
    lonBin := ""
    latBin := ""
    for i := 0; i < 192; i += 3 {
        tmBin += string(binaryString[i])
        lonBin += string(binaryString[i+1])
        latBin += string(binaryString[i+2])
    }

    // Reverse the bit-shifting in ConvertUnixTimeToShiftedBinary()
    unshiftedTmBin := tmBin[35:] + tmBin[:35]
    tmUnix, _ := strconv.ParseInt(unshiftedTmBin, 2, 64)
    tm = time.Unix(tmUnix, 0)

    lon = binaryStringToFloat64(lonBin)
    lat = binaryStringToFloat64(latBin)

    // the remainder are trailingBits we've appended to the end of the zOrderIndex
    trailingBits = binaryString[192:]

    return tm, lat, lon, trailingBits
}

// Helper function to convert binary string to float64
func binaryStringToFloat64(binaryStr string) float64 {
    // Flip the sign bit back
    binaryStr = "0" + binaryStr[1:]

    // Convert binary string to uint64
    bits, _ := strconv.ParseUint(binaryStr, 2, 64)

    // Convert uint64 to float64
    floatValue := math.Float64frombits(bits)

    // Subtract 2000 to reverse the addition in mapFloatToSortableBinaryString
    return floatValue - 2000
}
