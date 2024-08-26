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

func ConvertUnixTimeToBinary(unixTime int64) string {
    // Convert to binary without padding
    binaryStr := strconv.FormatInt(unixTime, 2)

    // Right-pad with zeros to ensure 64-bit length
    paddedBinaryStr := binaryStr + strings.Repeat("0", 64 - len(binaryStr))
    log.Println("startTime unix: ", unixTime)
    log.Println("startTime binaryStr: ", paddedBinaryStr)

    return paddedBinaryStr
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

    // indexCreationTimeBin := ConvertUnixTimeToBinary(indexCreationTime.Unix())
    tmBin := ConvertUnixTimeToBinary(tmUnix)
    log.Println("tmBin: ", tmBin)

    log.Println("lon: ", lon)
    log.Println("lat: ", lat)

    log.Println("middle negative lon", -20)

    _ = mapFloatToSortableBinaryString(10)

    log.Println("positive lon", 10)

    _ = mapFloatToSortableBinaryString(-20)


    lonSortableBinStr := mapFloatToSortableBinaryString(lon)
    latSortableBinStr := mapFloatToSortableBinaryString(lat)


    // Convert sortable integers to binary string
    // lonBin := fmt.Sprintf("%064b", lonSortableInt)
    // latBin := fmt.Sprintf("%064b", latSortableInt)

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
        zIndexBin += startTimeBin[i : i+1]
        zIndexBin += lonSortableBinStr[i : i+1]
        zIndexBin += latSortableBinStr[i : i+1]
    }

    // log.Println("zIndexBin: ", zIndexBin)

    // // Convert binary string to byte slice
    // zIndexBytes := make([]byte, 8)
    // for i := 0; i < len(zIndexBin) && i < 64; i += 8 {
    //     b, _ := strconv.ParseUint(zIndexBin[i:i+8], 2, 8)
    //     zIndexBytes[i/8] = byte(b)
    // }

    if indexType == "min" {
        // appendedBytes := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}
        // log.Println("MIN appendedBytes: ", appendedBytes)
        // zIndexBytes = append(zIndexBytes, appendedBytes...)
        // log.Println("zIndexBytes AFTER MIN append: ", zIndexBytes)

        zIndexBin += "0000000000000000000000000000000000000000000000000000000000000001"
    } else if indexType == "max" {
        // appendedBytes := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
        // log.Println("MAX appendedBytes: ", appendedBytes)
        // zIndexBytes = append(zIndexBytes, appendedBytes...)
        // log.Println("zIndexBytes AFTER MAX append: ", zIndexBytes)
        zIndexBin += "1111111111111111111111111111111111111111111111111111111111111111"
    } else {
        // Append index creation time as 8 bytes
        // appendedBytes := make([]byte, 8)
        // binary.BigEndian.PutUint64(appendedBytes, uint64(indexCreationTime.Unix()))
        // log.Println("IDX TIME appendedBytes: ", appendedBytes)
        // zIndexBytes = append(zIndexBytes, appendedBytes...)
        // log.Println("zIndexBytes IDX TIME append: ", zIndexBytes)
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
        // For negative values, flip all bits
        flippedBits := ""
        for _, bit := range binaryStr {
            if bit == '0' {
                flippedBits += "1"
            } else {
                flippedBits += "0"
            }
        }
        log.Println(">>>> flippedBits:", flippedBits)
        return flippedBits
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


// curl -X POST -H 'Content-Type: application/json' -d '{"name": "DC Bocce Ball Semifinals", "description": "Join us for the thrilling semifinals of the DC Bocce Ball Championship! Witness top-tier bocce action as teams compete for a spot in the finals. Enjoy refreshments, meet fellow bocce enthusiasts, and experience the excitement of this classic Italian game in the heart of DC. Whether you are a seasoned player or a curious spectator, this event promises an unforgettable evening of skill, strategy, and fun!", "datetime": "2024-07-15T18:30:00Z", "address": "National Mall, Washington, DC", "zip_code": "20001", "country": "USA", "latitude": 38.8951, "longitude": -77.0364}' https://8j5aj6o6v8.execute-api.us-east-1.amazonaws.com/api/event

// curl -X POST -H 'Content-Type: application/json' -d '{"name": "World Trivia Night Semifinals @ NYC", "description": "Calling all trivia buffs! The World Trivia Night Championship reaches its penultimate stage in the Big Apple. Teams from around the globe will battle it out in a test of knowledge spanning history, pop culture, science, and more. With high stakes and fierce competition, this event promises to be an intellectual spectacle. Join us for an evening of brain-teasing questions, international camaraderie, and the chance to witness trivia history in the making!", "datetime": "2024-08-22T19:00:00Z", "address": "Gotham Hall, 1356 Broadway, New York, NY", "zip_code": "10018", "country": "USA", "latitude": 40.6925, "longitude": -74.1687}' https://8j5aj6o6v8.execute-api.us-east-1.amazonaws.com/api/event


// curl -X POST -H 'Content-Type: application/json' -d '{"name": "Denver Karaoke League FINAL SHOWDOWN", "description": "Get ready for the ultimate sing-off at the Denver Karaoke League FINAL SHOWDOWN! After months of fierce competition, the top performers will take the stage to battle for the title of Denver Karaoke Champion. Expect show-stopping performances, surprise guest judges, and an electrifying atmosphere as contestants give it their all. Whether you are a participant or a spectator, this night promises unforgettable entertainment and the crowning of a new karaoke royalty!", "datetime": "2024-10-05T20:00:00Z", "address": "Grizzly Rose, 5450 N Valley Hwy, Denver, CO", "zip_code": "80216", "country": "USA", "latitude": 39.772896, "longitude": -105.07766}' https://8j5aj6o6v8.execute-api.us-east-1.amazonaws.com/api/event
