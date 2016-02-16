package main

import (
	log "bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/sirupsen/logrus"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/spf13/viper"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/koyachi/go-nude"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/disintegration/imaging"
	"image"
	"bytes"
	"text/template"
	"os"
	"math/rand"
	"image/color"
	"io/ioutil"
	"time"
	"strings"
	"fmt"
	"strconv"
	"path/filepath"
)

// recursively creates all dirs for fn
func createTargetDirs(fn string) {
	path, _ := filepath.Split(fn)
	os.MkdirAll(path, 0777)
}

// returns a random float32
func randomInt(min, max int) float32 {
	rand.Seed(time.Now().UTC().UnixNano())
	return float32(rand.Intn(max-min) + min)
}

// check if given fname exists already
func fileExists(fname string) bool {
	if _, err := os.Stat(fname); err == nil {
		return true
	}
	return false
}

// decides if an image should be considered blank (dark/white)
func isBlankImage(img image.Image) bool {
	blankPixels = 0
	allPixels = 0
	img = imaging.AdjustFunc(img, countBlankPixels)
	blankPercent := blankPixels / (allPixels / 100)
	// log.Debugf("image is %d percent black", blackPercent)
	if blankPercent >= 85 {
		log.Warnf("image is %d percent black, dropping frame", blankPercent)
		return true
	}
	return false
}

// count pixels which are white and/or black and writes them in blankPixels
func countBlankPixels(c color.NRGBA) color.NRGBA {
	//use 55?
	if int(c.R) < 50 && int(c.G) < 50 && int(c.B) < 50 {
		blankPixels = blankPixels + 1
	} else if int(c.R) > 200 && int(c.G) > 200 && int(c.B) > 200 {
		blankPixels = blankPixels + 1
	}

	allPixels = allPixels + 1

	return color.NRGBA{c.R, c.G, c.B, c.A}
}

// get font path for fontname
// searches in common font paths, bindata or absoute path
func getFont(f string) ([]byte, error) {
	if !strings.HasSuffix(f, ".ttf") {
		f = fmt.Sprintf("%s.ttf", f)
	}
	if strings.Contains(f, "/") && strings.HasSuffix(f, ".ttf") {
		if _, err := os.Stat(f); err == nil {
			log.Infof("useing font: %s", f)
			return ioutil.ReadFile(f)
		}
	}
	fdirs := []string{"/Library/Fonts/", "/usr/share/fonts/", "./"}

	for _, dir := range fdirs {
		fpath := filepath.Join(dir, f)
		if _, err := os.Stat(fpath); err == nil {
			log.Infof("useing font: %s", fpath)
			return ioutil.ReadFile(fpath)
		}
	}
	log.Info("useing font: DroidSans.ttf")
	return Asset("DroidSans.ttf")
}

// takes a time based string 00:00:00 and converts it to milliseconds
func stringToMS(s string) int64 {
	x := strings.Split(s, ":")
	hh, _ := strconv.Atoi(x[0])
	mm, _ := strconv.Atoi(x[1])
	ss, _ := strconv.Atoi(x[2])

	end := (ss + (mm * 60) + (hh * 60 * 60)) * 1000
	return int64(end)
}


// wrapper for nudity detection
func isNudeImage(img image.Image) bool {
	isNude, err := nude.IsImageNude(img)
	if err != nil {
		log.Error(err)
		return false
	}
	// d := nude.NewDetector(img)
	// isNude, err := d.Parse()
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Printf("isNude = %v\n", isNude)
	// fmt.Printf("%s\n", d)
	return isNude
}

//takes a string "0,0,0" and returns the RGBA color
func getImageColor(s string, fallback []int) color.RGBA {
	colors := strings.Split(s, ",")
	var r, g, b int
	if len(colors) == 3 {
		r, _ = strconv.Atoi(strings.TrimSpace(colors[0]))
		g, _ = strconv.Atoi(strings.TrimSpace(colors[1]))
		b, _ = strconv.Atoi(strings.TrimSpace(colors[2]))
		log.Debugf("color %s converted to [%d %d %d]", s, r, g, b)
	} else {
		log.Warnf("error converting %s to a valid color, useing fallback color: %v", s, fallback)
		r, g, b = fallback[0], fallback[1], fallback[2]
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// used to construct a save path based on given file info
type FileInfo struct {
	Name  string
	Ext   string
	Path  string
	Count string
}

// increment savePath as long as there is a file present
func increamentSavePath(filename string, c int) string {
	fname := filename
	counter := c
	for fileExists(fname) {
		log.Debugf("image already existing at: %s")
		fname = constructSavePath(filename, counter)
		counter++
	}
	return fname
}

// constructs the save path based on filename and counter
func constructSavePath(filename string, c int) string {
	if viper.GetString("filename") == "%s.jpg" {
		return fmt.Sprintf(viper.GetString("filename"), filename)
	}

	out := viper.GetString("filename")

	fx := FileInfo{}
	fx.Path, fx.Name = filepath.Split(filename)
	fx.Ext = filepath.Ext(filename)
	fx.Name = strings.Replace(fx.Name, fx.Ext, "", -1)
	fx.Count = fmt.Sprintf("%02d", c)
	if c > 0 {
		if !strings.Contains(out, "{{.Count}}") {
			out = strings.Replace(out, ".jpg", "-{{.Count}}.jpg", 1)
		}
	}

	t := template.Must(template.New("filepath").Parse(out))
	buf := new(bytes.Buffer)
	t.Execute(buf, &fx)

	if buf.String() == "" {
		return strings.Replace(filename, fx.Ext, ".jpg", -1)
	}
	return buf.String()
}

//gets a filename (string) and returns the absolute path to save the image to...
func getSavePath(filename string, c int) string {
	fname := constructSavePath(filename, c)

	if viper.GetBool("skip_existing") && fileExists(fname) {
		return fname
	}

	counter := c
	for fileExists(fname) && !viper.GetBool("overwrite") {
		//log.Debugf("image already existing at: %s and overwrite is disabled", fname)
		counter++
		fname = constructSavePath(filename, counter)
	}
	return fname
}
