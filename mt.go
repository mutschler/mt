package main

import (
	"flag"
	"fmt"
	"github.com/opennota/screengen"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"time"
	// "path"
	// "path/filepath"
	// "strings"
	"github.com/disintegration/imaging"
	// "github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func writeImage(img image.Image, fn string) {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	err = jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	if err != nil {
		fmt.Fprintf(os.Stderr, "JPEG encoding error: %v\n", err)
		os.Exit(1)
	}
}

// generates screenshots and returns a list of images
func GenerateScreenshots(fn string) []image.Image {
	var thumbnails []image.Image
	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading video file: %v\n", err)
		os.Exit(1)
	}
	defer gen.Close()

	//skip 4 minutes of duration to remove intro and credits
	inc := (gen.Duration - 240000) / int64(viper.GetInt("images"))
	d := inc
	for i := 0; i < viper.GetInt("images"); i++ {
		img, err := gen.Image(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't generate screenshot: %v\n", err)
			os.Exit(1)
		}

		// fn := filepath.Join("/", fmt.Sprintf(viper.GetString("filename"), fn, i))
		//writeImage(img, fn)
		fmt.Printf("generating screenshot %02d/%02d at %s\n", i, viper.GetInt("images"), fmt.Sprintf(time.Unix(d/1000, 0).UTC().Format("15:04:05")))
		thumb := imaging.Resize(img, viper.GetInt("width"), 0, imaging.Lanczos)
		thumbnails = append(thumbnails, thumb)
		d += inc
	}

	return thumbnails
}

func makeContactSheet(thumbs []image.Image, fn string) {
	fmt.Println("Composing Contact Sheet")
	imgWidth := thumbs[0].Bounds().Dx()
	imgHeight := thumbs[0].Bounds().Dy()

	imgPerRow := 4
	imgRows := len(thumbs) / 4

	// create a new blank image
	// fmt.Printf("Single Image: %d x %d", imgWidth, imgHeight)
	// fmt.Printf("New Image: %d x %d", 400*4, imgHeight*imgPerRow)

	dst := imaging.New(imgWidth*imgPerRow, imgHeight*imgRows, color.NRGBA{0, 0, 0, 0})
	x := 0
	curRow := 0
	// paste thumbnails into the new image side by side
	for _, thumb := range thumbs {
		//fmt.Print(i)
		if x >= imgPerRow {
			x = 0
			curRow = curRow + 1
		}

		// fmt.Printf("%d x %d\n", x*imgWidth, curRow*imgHeight)
		dst = imaging.Paste(dst, thumb, image.Pt(x*imgWidth, curRow*imgHeight))
		x = x + 1
	}

	// save the combined image to file
	err := imaging.Save(dst, fn)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Saved to %s", fn)
}

func main() {
	viper.SetDefault("images", 24)
	viper.SetDefault("width", 400)
	viper.SetDefault("filename", "%s.jpg")

	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// movie := "/Volumes/Datengrab/Serien/Archer/Staffel 01/Archer.s01e01.Der.Maulwurf.720p.BluRay.mkv"
	for _, movie := range flag.Args() {
		fmt.Printf("generating contact sheet for %s\n", movie)
		thumbs := GenerateScreenshots(movie)
		makeContactSheet(thumbs, fmt.Sprintf(viper.GetString("filename"), movie))
	}

}
