package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

func main() {
	fmt.Println("Where tf is Waldo?")

	dat, err := os.ReadFile("./input/where-waldo.jpeg")

	if err != nil {
		panic("No waldo image!")
	}

	// Oh no, we need to decode the JPEG image

	// We'll take this as reference for the image syntax https://en.wikipedia.org/wiki/JPEG#Syntax_and_structure

	// JPEG image must start with SOI bytes 0xFF, 0xD8
	if fmt.Sprintf("%x", dat[0]) != "ff" || fmt.Sprintf("%x", dat[1]) != "d8" {
		panic("JPEG Image has invalid SOI")
	} else {
		fmt.Println("Valid JPEG SOI")
	}

	// Next sequence of bytes are ff e0 0 10 4a 46 49 46 0 1 1 0 0 1 0 1 0 0
	// But what does `ff e0` represent?
	// From https://www.w3.org/Graphics/JPEG/jfif3.pdf
	// Apparently it's called an APP0 marker.
	// AND apparently JPEGs are a lie and most files are actually JFIFs?

	app0Marker, err := parseAPP0Marker(&dat, 2)

	if err != nil {
		panic(err)
	}

	// Continue from SOI length + APPO marker itself + length of JFIF APP0 marker segment
	idx := 2 + 2 + int(app0Marker.length)

	// Next comes:
	// 2 quantization tables (ff db)
	// 1 start of frame (ff c2)
	// 2 huffman tables (ff c4)

	qTable1, err := parseQtzTable(&dat, idx)

	// Pain point: If I try to use qTable1 before checking the error,
	// the compiler doesn't tell me that I shouldn't do that
	if err != nil {
		panic(err)
	}

	idx += int(qTable1.length) + 2

	qTable2, err := parseQtzTable(&dat, idx)

	if err != nil {
		panic(err)
	}

	idx += int(qTable2.length) + 2

	startFrame, err := parseStartingFrame(&dat, idx)

	if err != nil {
		panic(err)
	}

	idx += int(startFrame.length) + 2

	fmt.Printf("start frame %+v\n", startFrame)

	fmt.Print("cont: ")
	for i := 0; i < 100; i++ {
		fmt.Printf("%x ", dat[idx+i])
	}
	fmt.Print("\n")
}

type JFIFDensity byte

const (
	NoDensity JFIFDensity = 0
	Ppi       JFIFDensity = 1
	Ppcm      JFIFDensity = 2
)

type APP0Marker struct {
	length       uint16
	jfifVersion  [2]byte
	densityUnits JFIFDensity
	xDensity     uint16
	yDensity     uint16
	xThumbnail   uint8
	yThumbnail   uint8
}

type QuantizationTable struct {
	length      uint16
	precision   uint8
	destination uint8
	data        [8][8]int
}

type StartingFrame struct {
	length          uint16
	samplePrecision byte
	width           uint16
	height          uint16
	numComponents   int
	componentSpecs  []ComponentSpec
}

type ComponentSpec struct {
	componentType   byte
	samplingFactors byte
	qTable          byte
}

// Apparently, in Go function values and returns are copies by default.
// That's why it's important to pass and return pointers.
func parseAPP0Marker(data *[]byte, idx int) (*APP0Marker, error) {
	marker := APP0Marker{}

	// Check that first and second bytes are for APP0 marker

	// Kinda weird that go doesn't have a method to get a list element
	// that returns the value or error
	firstByte := (*data)[idx]
	secondByte := (*data)[idx+1]
	if firstByte != 0xff || secondByte != 0xe0 {
		return nil, errors.New("JPEG image has invalid APP0 marker")
	}
	idx += 2

	// Now what's inside an APP0 marker?
	// Reference https://en.wikipedia.org/wiki/JPEG_File_Interchange_Format#JFIF_extension_APP0_marker_segment

	// According to https://en.wikipedia.org/wiki/JPEG_File_Interchange_Format#File_format_structure
	// We need to parse the lenght in big endian
	marker.length = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	// "JFIF" in ASCII, followed by a null byte
	expectedIdentifier := []uint8{0x4a, 0x46, 0x49, 0x46, 0x00}

	for i := 0; i < len(expectedIdentifier)-1; i++ {
		if (*data)[idx] != expectedIdentifier[i] {
			fmt.Printf("Expected: %x. Actual %x\n", byte(expectedIdentifier[i]), (*data)[idx])
			return nil, errors.New("invalid JFIF identifier")
		}
		idx++
	}
	idx++

	// Get the version (two bytes for major and minor)
	marker.jfifVersion = [2]byte{(*data)[idx], (*data)[idx+1]}
	idx += 2

	// Pain point: Passing an invalid value to cast does not error or panic
	marker.densityUnits = JFIFDensity((*data)[idx])
	idx++

	// Get density for x and y
	marker.xDensity = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	marker.yDensity = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	// Densities can't be 0
	if marker.xDensity == 0 {
		return nil, errors.New("horizontal density is zero")
	}

	if marker.yDensity == 0 {
		return nil, errors.New("vertical density is zero")
	}

	// Get pixel counts for x and y
	// Interesting: Uint16 needs 2 bytes! No compiler check for that tho :c
	marker.xThumbnail = uint8((*data)[idx])
	idx++

	marker.yThumbnail = uint8((*data)[idx])
	idx++

	// No embedded thumbnail in image
	if marker.xThumbnail == 0 || marker.yThumbnail == 0 {
		return &marker, nil
	}

	// TODO(maybe never): get data from embedded thumbnail
	return &marker, nil
}

func parseQtzTable(data *[]byte, idx int) (*QuantizationTable, error) {
	table := QuantizationTable{}

	// Check that the first and second bytes are the quantization table markers "ff cb"
	firstByte := (*data)[idx]
	secondByte := (*data)[idx+1]
	if firstByte != 0xff || secondByte != 0xdb {
		return nil, errors.New("JPEG image has invalid QT marker")
	}
	idx += 2

	// Next 2 bytes indicate the length of the marker
	table.length = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	// Next byte has the precision and destination id of the quant table
	precisionAndIdByte := (*data)[idx]

	// First 4 bits are precision
	table.precision = (precisionAndIdByte >> 4) & 0x0f

	// Last 4 bits are destination
	table.destination = precisionAndIdByte & 0x0f

	idx++

	// Now comes the data (minus 3 for length and precision bytes)
	for i := 0; i < int(table.length-3); i++ {
		row := int(i / 8)
		col := i - (row)*8
		table.data[row][col] = int((*data)[idx+i])
	}

	return &table, nil
}

func parseStartingFrame(data *[]byte, idx int) (*StartingFrame, error) {
	frame := StartingFrame{}

	// Check that first two bytes represent a SOF marker
	firstFrame := (*data)[idx]
	secondFrame := (*data)[idx+1]

	if firstFrame != 0xff || secondFrame != 0xc2 {
		return nil, errors.New("invalid SOF marker")
	}

	idx += 2

	frame.length = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	frame.samplePrecision = (*data)[idx]
	idx++

	frame.height = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	frame.width = binary.BigEndian.Uint16((*data)[idx : idx+2])
	idx += 2

	frame.numComponents = int((*data)[idx])
	idx++

	for i := 0; i < frame.numComponents; i++ {
		frame.componentSpecs = append(frame.componentSpecs, ComponentSpec{
			componentType:   (*data)[idx],
			samplingFactors: (*data)[idx+1],
			qTable:          (*data)[idx+2],
		})
		idx += 3
	}

	return &frame, nil
}
