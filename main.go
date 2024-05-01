package main

import (
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

	// Parse APP0 marker.
	// Bytes #2 and #3 are actually just the marker. So let's make sure it's still there
	if fmt.Sprintf("%x", dat[2]) != "ff" || fmt.Sprintf("%x", dat[3]) != "e0" {
		panic("JPEG Image has invalid APP0 marker")
	} else {
		fmt.Println("Valid APP0 marker")
	}

	// Now what's inside an APP0 marker?
	/*
	 From https://en.wikipedia.org/wiki/JPEG_File_Interchange_Format#JFIF_extension_APP0_marker_segment
	 1. The length (excluding the marker)
	 2. The "JFXX" identifier (terminated with a null byte)
	 3. The thumbnail format (JPEG, 1 byte/px palettized format or 1byte/px RGB format) (???)
	 4. The thumbnail data (depends on the format)
	*/
}
