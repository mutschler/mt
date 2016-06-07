// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
// Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

// A tool that generates thumbnail grids from video files with the help of ImageMagick's convert.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mutschler/mt/Godeps/_workspace/src/github.com/opennota/screengen"
)

const (
	thWidth   = 280
	thSpacing = 14
)

var (
	n                = flag.Int("n", 33, "Number of thumbnails")
	thumbnailsPerRow = flag.Int("thumbnails-per-row", 3, "Thumbnails per row")
	output           = flag.String("o", "", "Output file (default: video file name + .jpg)")
	quality          = flag.Int("quality", 85, "Output image quality")
	font             = flag.String("font", "LiberationSans", "Normal font face")
	fontBold         = flag.String("font-bold", "LiberationSansB", "Bold font face")
	comment          = flag.String("comment", "", "Comment")
)

type Image struct {
	time     int64
	filename string
}

func divRoundHalfUp(a, b int64) int64 {
	return int64(math.Floor(float64(a)/float64(b) + 0.5))
}

func fileSizeHuman(name string) (string, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d Mb", divRoundHalfUp(fi.Size(), 1024*1024)), nil
}

func writePng(img image.Image) (string, error) {
	f, err := ioutil.TempFile("", "screengen")
	if err != nil {
		return "", err
	}

	err = png.Encode(f, img)
	if err != nil {
		f.Close()
		return "", err
	}

	err = f.Close()
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func divRoundUp(a, b int) int {
	c := a / b
	if a%b > 0 {
		c++
	}
	return c
}

func ms2String(ms int64) string {
	s := ms / 1000
	h := s / 60 / 60
	s -= h * 60 * 60
	m := s / 60
	s -= m * 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

var escaper = strings.NewReplacer("'", `\'`, `\`, `\\`)

func makeThumbnailGrid(g *screengen.Generator) error {
	fileSize, err := fileSizeHuman(g.Filename)
	if err != nil {
		return fmt.Errorf("can't get file size: %v", err)
	}

	thHeight := int(float64(g.Height()) * thWidth / float64(g.Width()))
	images := make([]Image, 0, *n)
	inc := g.Duration / int64(*n)
	d := inc / 2
	for i := 0; i < *n; i++ {
		img, err := g.ImageWxH(d, thWidth, thHeight)
		if err != nil {
			return fmt.Errorf("can't extract image: %v", err)
		}

		fn, err := writePng(img)
		if err != nil {
			return fmt.Errorf("can't write thumbnail: %v", err)
		}
		defer os.Remove(fn)

		images = append(images, Image{
			time:     d,
			filename: fn,
		})

		d += inc
	}

	numRows := divRoundUp(len(images), *thumbnailsPerRow)
	w := *thumbnailsPerRow*thWidth + (*thumbnailsPerRow+1)*thSpacing
	h := numRows*thHeight + (numRows+1)*thSpacing
	const xOffset = 80
	const lineHeight = 16
	args := []string{
		"(",
		"-size", fmt.Sprintf("%dx%d", w, 128),
		"xc:white",
		"-fill", "black",

		"-font", *fontBold,
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2, "Filename:"),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2+lineHeight, "Size:"),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2+lineHeight*2, "Duration:"),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2+lineHeight*3, "Resolution:"),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2+lineHeight*4, "Video:"),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2+lineHeight*5, "Audio:"),

		"-font", *font,
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing+xOffset, thSpacing*2, escaper.Replace(filepath.Base(g.Filename))),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing+xOffset, thSpacing*2+lineHeight, fileSize),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing+xOffset, thSpacing*2+lineHeight*2, ms2String(g.Duration)),
		"-draw", fmt.Sprintf("text %d,%d '%dx%d'", thSpacing+xOffset, thSpacing*2+lineHeight*3, g.Width(), g.Height()),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing+xOffset, thSpacing*2+lineHeight*4, g.VideoCodecLongName),
		"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing+xOffset, thSpacing*2+lineHeight*5, g.AudioCodecLongName),
	}

	if *comment != "" {
		args = append(args,
			"-font", *fontBold,
			"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing, thSpacing*2+lineHeight*6, "Comment:"),
			"-font", *font,
			"-draw", fmt.Sprintf("text %d,%d '%s'", thSpacing+xOffset, thSpacing*2+lineHeight*6, escaper.Replace(*comment)),
		)
	}

	args = append(args,
		")",

		"(",
		"-size", fmt.Sprintf("%dx%d", w, h),
		"xc:white",
		"-gravity", "northwest",
		"-font", *font,
	)

	x := 0
	y := 0
	for _, img := range images {
		time := ms2String(img.time)
		args = append(args,
			"(",
			img.filename,
			"-fill", "black",
			"-draw", fmt.Sprintf("text %d,%d '%s'", 5, 5, time),
			"-fill", "white",
			"-draw", fmt.Sprintf("text %d,%d '%s'", 6, 6, time),
			")",
			"-geometry", fmt.Sprintf("+%d+%d", x*(thWidth+thSpacing)+thSpacing, y*(thHeight+thSpacing)+thSpacing),
			"-composite",
		)

		x++
		if x == *thumbnailsPerRow {
			x = 0
			y++
		}
	}

	args = append(args, ")", "-append", "-quality", strconv.Itoa(*quality), *output)

	cmd := exec.Command("convert", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: screengen [options] videofile")
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		return
	}

	filename := flag.Arg(0)
	if *output == "" {
		*output = filepath.Base(filename) + ".jpg"
	}

	g, err := screengen.NewGenerator(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = makeThumbnailGrid(g)
	if err != nil {
		log.Fatal(err)
	}
}
