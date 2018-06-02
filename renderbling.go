package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

/*
 * Lots of things are hard coded, since it's assumed that this will be converted for use per-badge.
 * This assumes a 128x128 pixel LED display, the same as on the Bender, Joco, and Trans-Ionospheric badges
 */

/* The size of the master badge image */
const masterxsz = 520
const masterysz = 565

/*
 * This is an attempt to abstract the LCD bling display size, but it's only ever been tested with
 * 128x128 pixels
 */
const anixsz = 128
const aniysz = 128

/* Location of the 128x128 LCD display within the master image */
const lcdorgx = 200
const lcdorgy = 200

/* LED Locations */
type location struct {
	x uint
	y uint
}

/*
 * Locations of the RGB LEDs
 *
 * These are the locations of the center of each of the LEDs on the badge,
 * mapped to the order that they appear in the .RGB file
 */
var ll = []location{
	location{x: 163, y: 127}, /*  0 */
	location{x: 218, y: 127}, /*  1 */
	location{x: 263, y: 127}, /*  2 */
	location{x: 309, y: 127}, /*  3 */
	location{x: 361, y: 127}, /*  4 */
	location{x: 410, y: 277}, /*  5 */
	location{x: 410, y: 319}, /*  6 */
	location{x: 391, y: 358}, /*  7 */
	location{x: 148, y: 508}, /*  8 */
	location{x: 225, y: 508}, /*  9 */
	location{x: 299, y: 508}, /* 10 */
	location{x: 377, y: 508}, /* 11 */
	location{x: 138, y: 358}, /* 12 */
	location{x: 115, y: 319}, /* 13 */
	location{x: 115, y: 277}, /* 14 */
}

/*
 * The number of bytes in an LED frame in the .RGB files.
 * There are 3 bytes per LED for RGB.
 * For all the supported badges, the frame length is 60, even though
 * fewer than 20 LEDs are used.
 */
const lflen = 60

/* The default number of frames to render */
const defframes = 200

/* End of customization */

/*
 We're converting from 8 bits per color to 565, so we shift
 to remove the low order bits
*/
func rgb2pixel(r uint8, g uint8, b uint8) uint16 {
	var rt uint16
	var gt uint16
	var bt uint16
	var all uint16
	rt = uint16(r) >> 3
	gt = uint16(g) >> 2
	bt = uint16(b) >> 3
	all = (rt << 11) | (gt << 5) | bt
	return (all)
}

func writepixel(x uint, y uint, value uint16, image [][]uint16) {
	image[x][y] = value
}

func readSome(f *os.File, len int) ([]uint16, error) {
	/* read a frame from the file. If we hit EOF, reset to start at beginning */
	frame := make([]uint16, len)
	err := binary.Read(f, binary.BigEndian, frame)
	if err == io.EOF {
		_, err = f.Seek(0, 0)
		if err != nil {
			log.Fatal(err)
		}
		err = binary.Read(f, binary.BigEndian, frame)
		if err != nil {
			log.Fatal(err)
		}
	}
	return frame, err
}

func readRGB(f *os.File) ([]uint8, error) {
	frame := make([]byte, lflen)
	_, err := f.Read(frame)
	if err == io.EOF {
		_, err = f.Seek(0, 0)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Read(frame)
		if err != nil {
			log.Fatal(err)
		}
	}
	return frame, err
}

func usage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("%s -master=<filename> -bling=<filename> -rgb=<filename> -out=<filename> [-frames=<numframes>]\n\n", filepath.Base(os.Args[0]))
	fmt.Printf("Where:    master  = filename for background image in .RAW format [REQUIRED]\n")
	fmt.Printf("            bling = filename for LCD animation in .RAW format [REQUIRED]\n")
	fmt.Printf("              rgb = filename for LED bling in .RGB format [REQUIRED]\n")
	fmt.Printf("              out = output filename [REQUIRED]\n")
	fmt.Printf("           frames = number of frames to render (defaults to %d)\n", defframes)
	return
}

func main() {

	var masterimagefile = flag.String("master", "", "Master Image File for badge background")
	var blingfile = flag.String("bling", "", "Bling Animation to display in LCD panel")
	var rgbfile = flag.String("rgb", "", "LED .RGB file to use for LED displays")
	var outfilename = flag.String("out", "", "Output File Name")
	var numframes = flag.Int("frames", defframes, "Number of output frames to render")

	flag.Parse()

	if *masterimagefile == "" || *blingfile == "" || *rgbfile == "" || *outfilename == "" {
		usage()
		os.Exit(1)
	}

	masterimage := make([][]uint16, masterysz)
	for i := range masterimage {
		masterimage[i] = make([]uint16, masterxsz)
	}

	aniimage := make([][]uint16, aniysz)
	for i := range aniimage {
		aniimage[i] = make([]uint16, anixsz)
	}

	file, err := os.Open(*masterimagefile)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	/* Read the master image */
	for yp := 0; yp < masterysz; yp++ {
		for xp := 0; xp < masterxsz; xp++ {
			err := binary.Read(file, binary.BigEndian, &masterimage[yp][xp])
			if err != nil {
				log.Println(err)
				break
			}
		}
	}

	/* Open the input LCD bling */
	anifile, err := os.Open(*blingfile)
	defer anifile.Close()
	if err != nil {
		log.Fatal(err)
	}

	/* Open the LED RGB Bling */
	rgbf, err := os.Open(*rgbfile)
	defer rgbf.Close()
	if err != nil {
		log.Fatal(err)
	}

	/* Open the output file */
	outfile, err := os.Create(*outfilename)
	defer outfile.Close()
	if err != nil {
		log.Fatal(err)
	}

	for frame := 0; frame < *numframes; frame++ {
		for row := 0; row < aniysz; row++ {
			rowdata, err := readSome(anifile, anixsz)
			if err != nil {
				log.Fatal(err)
			}
			for col, val := range rowdata {
				masterimage[row+lcdorgy][col+lcdorgx] = val
			}
		}
		ledval, err := readRGB(rgbf)
		if err != nil {
			log.Fatal(err)
		}

		didx := 0
		for _, pos := range ll {
			var red uint8
			var green uint8
			var blue uint8
			red = ledval[didx]
			green = ledval[didx+1]
			blue = ledval[didx+2]
			pxval := rgb2pixel(red, green, blue)
			for x := pos.x - 7; x <= pos.x+7; x++ {
				for y := pos.y - 7; y <= pos.y+7; y++ {
					masterimage[y][x] = pxval
				}
			}
			didx += 3
		}

		for y := 0; y < masterysz; y++ {
			err := binary.Write(outfile, binary.BigEndian, masterimage[y])
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
