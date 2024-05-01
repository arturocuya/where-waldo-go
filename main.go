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

	fmt.Printf("JFIF APP0 Marker %+v\n", app0Marker)

	// Continue from SOI length + APPO marker itself + length of JFIF APP0 marker segment
	idx := 2 + 2 + app0Marker.length

	for i := idx; i < 100; i++ {
		fmt.Printf("%x ", dat[i])
	}
	fmt.Print("\n")

	// Next comes a quantization table (marked by ff db). What's this? How to parse?
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
