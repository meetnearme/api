package indexing

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

func ConvertUnixTimeToBinary(unixTime int64) string {
    // Convert to binary without padding
    binaryStr := strconv.FormatInt(unixTime, 2)

    // Right-pad with zeros to ensure 64-bit length
    paddedBinaryStr := binaryStr + strings.Repeat("0", 64-len(binaryStr))

    log.Println("startTime binaryStr: ", paddedBinaryStr)

    return paddedBinaryStr
}

func CalculateZOrderIndex(startTime time.Time, lat, lon float64, indexType string) ([]byte, error) {
    // Get current timestamp as index creation time
    indexCreationTime := time.Now().UTC()
    startTimeUnix := startTime.Unix()
    // Convert dimensions to binary representations

    // indexCreationTimeBin := ConvertUnixTimeToBinary(indexCreationTime.Unix())
    startTimeBin := ConvertUnixTimeToBinary(startTimeUnix)
    log.Println("startTimeBin: ", startTimeBin)

    // Map floating point values to sortable unsigned integers
    // lonSortableInt := mapFloatToSortableInt32(lon)
    // latSortableInt := mapFloatToSortableInt32(lat)

    lonSortableInt := mapFloatToSortableInt64(float64(lon))
    latSortableInt := mapFloatToSortableInt64(float64(lat))


    // Convert sortable integers to binary string
    lonBin := fmt.Sprintf("%064b", lonSortableInt)
    latBin := fmt.Sprintf("%064b", latSortableInt)

    log.Println("lonBin: ", lonBin)
    log.Println("latBin: ", latBin)
    log.Println("startTimeBin: ", startTimeBin)


    // latBin = "1111111111111111111111111111111111111111111111111111111111111111"

    // lonBin = "0000000000000000000000000000000000000000000000000000000000000000"

    // startTimeBin = "0000000000000000000000000000000000000000000000000000000000000000"

    // min startTimeBin
    // 1100110110010001000000101111010000000000000000000000000000000000
    // max startTimeBin
    // 1001000101110000001000011011110100000000000000000000000000000000

    // min lonBin
    // 0011111110101010010110111111111110001000010001010110101001110010
    // max lonBin
    // 0011111110110001000111001101000011101111011101010010101100011001

    // min latBin
    // 0100000000111111010010111001010111111011001010000001011100000000
    // max latBin
    // 0100000001001001000010110111110011000010011010111111010011000000


    // Interleave binary representations
    var zIndexBin string

    for i := 0; i < 64; i++ {
        zIndexBin += lonBin[i : i+1]
        zIndexBin += latBin[i : i+1]
        zIndexBin += startTimeBin[i : i+1]
    }

    log.Println("zIndexBin: ", zIndexBin)

    // Convert binary string to byte slice
    zIndexBytes := make([]byte, 8)
    for i := 0; i < len(zIndexBin) && i < 64; i += 8 {
        b, _ := strconv.ParseUint(zIndexBin[i:i+8], 2, 8)
        zIndexBytes[i/8] = byte(b)
    }

    if indexType == "min" {
        appendedBytes := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}
        log.Println("MIN appendedBytes: ", appendedBytes)
        zIndexBytes = append(zIndexBytes, appendedBytes...)
    } else if indexType == "max" {
        appendedBytes := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
        log.Println("MAX appendedBytes: ", appendedBytes)
        zIndexBytes = append(zIndexBytes, appendedBytes...)
    } else {
        // Append index creation time as 8 bytes
        appendedBytes := make([]byte, 8)
        binary.BigEndian.PutUint64(appendedBytes, uint64(indexCreationTime.Unix()))
        log.Println("IDX TIME appendedBytes: ", appendedBytes)
        zIndexBytes = append(zIndexBytes, appendedBytes...)
    }



// START ORIGINAL CODE

    // Convert binary string to byte slice
    // zIndexBytes := make([]byte, len(zIndexBin)/8)
    // for i := 0; i < len(zIndexBin); i += 8 {
    //     b, _ := strconv.ParseUint(zIndexBin[i:i+8], 2, 8)
    //     zIndexBytes[i/8] = byte(b)
    // }

    // log.Println("zIndexBytes: ", zIndexBytes)
    // if indexType == "min" {
    //     appendedBytes := make([]byte, 8)
    //     binary.BigEndian.PutUint64(appendedBytes, 0)
    //     zIndexBytes = append(zIndexBytes, appendedBytes...)
    // } else if indexType == "max" {
    //     appendedBytes := make([]byte, 8)
    //     binary.BigEndian.PutUint64(appendedBytes, math.MaxUint64)
    //     zIndexBytes = append(zIndexBytes, appendedBytes...)
    // } else {
    //     // Append index creation time as bytes
    //     // indexCreationTimeBytes, _ := indexCreationTime.MarshalBinary()
    //     log.Println("indexCreationTimeBin: ", indexCreationTimeBin)
    //     // log.Println("indexCreationTimeBytes: ", indexCreationTimeBytes)
    //     log.Println("len indexCreationTimeBin: ", len(indexCreationTimeBin))
    //     zIndexBytes = append(zIndexBytes, indexCreationTimeBin...)
    // }

// END ORIGINAL CODE

    return zIndexBytes, nil
}


// Denver Karaoke League FINAL SHOWDOWN
// BSSSRQCLZaQaJILaViW1KDQH6Xo41SYQMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMTEwMDExMDExMDAwMTExMTExMDEwMDAxMTAwMTAwMA==

// Bocce Ball DC
// BSSSRQSJRbSKYRQacqW7iGHHVjw7o6zIMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMTEwMDExMDExMDAwMTExMTExMDEwMTAwMDEwMTAwMA==

// World Trivia NYC
// BSSSRQTCDTKCIaCJNhHp2Qc8iJ5x5tQTMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMTEwMDExMDExMDAwMTExMTExMDEwMDEwMDAwMTExMA==




// BigEndian
// [sst] |  +3548ms 2024/08/22 19:28:54 startTime binaryStr:  0000000000000000000000000000000001100110110001111110010111010110
// [sst] |  +3548ms 2024/08/22 19:28:54 startTime binaryStr:  0000000000000000000000000000000100100010110111111010011111010110


// LittleEndian

// [sst] |  +2102ms 2024/08/22 19:30:56 startTime binaryStr:  0000000000000000000000000000000001100110110001111110011001010000
// [sst] |  +2103ms 2024/08/22 19:30:56 startTime binaryStr:  0000000000000000000000000000000100100010110111111010100001010000
// [sst] |  Done in 2919ms


// 0000000000000000000000000000000001100110110001111110010111010110
// 0000000000000000000000000000000001100110110001111110011001010000

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
