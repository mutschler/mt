package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/disintegration/gift"
	"github.com/disintegration/imaging"
	"github.com/koyachi/go-nude"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

// decides if an image should be skipped based on settings
func skipImage(img image.Image) bool {

	if viper.GetBool("skip_blurry") {
		if isBluryImage(img) {
			return true
		}
	}

	if viper.GetBool("skip_blank") {
		if isBlankImage(img) {
			return true
		}
	}

	if viper.GetBool("sfw") {
		if isNudeImage(img) {
			return true
		}
	}

	return false

}

// decides if an image is to blury
func isBluryImage(img image.Image) bool {
	blur := 0
	g := gift.New(
		gift.Convolution(
			[]float32{
				-1, -1, -1,
				-1, 8, -1,
				-1, -1, -1,
			},
			false, false, false, 0.0),
	)
	img = imaging.Grayscale(img)
	dst := image.NewRGBA(g.Bounds(img.Bounds()))
	g.Draw(dst, img)
	pixels := 0
	for x := 0; x < dst.Bounds().Dx(); x++ {
		for y := 0; y < dst.Bounds().Dy(); y++ {
			_, _, b, _ := dst.At(x, y).RGBA()

			pixels = pixels + 1
			// only count blue channel < 4
			if int(b) < 2056 {
				blur = blur + 1
			}
		}
	}

	blurPercent := int((float32(blur) / float32(pixels)) * 100)
	if blurPercent >= viper.GetInt("blur_threshold") {
		log.Debugf("image is considered blurry (%d), dropping frame", blurPercent)
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
	if blankPercent >= viper.GetInt("blank_threshold") {
		log.Debugf("image is %d percent black, dropping frame", blankPercent)
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

	if s == "0" {
		return 0
	}

	x := strings.Split(s, ":")

	if len(x) != 3 {
		log.Warnf("unable to convert string '%s' into milliseconds string not in format hh:mm:ss", s)
		return 0
	}

	hh, _ := strconv.Atoi(x[0])
	mm, _ := strconv.Atoi(x[1])
	ss, _ := strconv.Atoi(x[2])
	ms := 0

	if strings.Contains(x[2], ".") {
		x = strings.Split(x[2], ".")
		ss, _ = strconv.Atoi(x[0])
		ms, _ = strconv.Atoi(x[1])
	}

	end := ((ss + (mm * 60) + (hh * 60 * 60)) * 1000) + ms
	return int64(end)
}

// wrapper for nudity detection
func isNudeImage(img image.Image) bool {
	isNude, err := nude.IsImageNude(img)
	if err != nil {
		log.Error(err)
		return false
	}
	if isNude {
		log.Debugf("image skipped because of nudity detection")
	}

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
		log.Debugf("image already existing at: %s", fname)
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

// saves the image to a temporary location and returns the path
func saveTempFile(img image.Image) string {
	if tmpDir == "" {
		tmpDir, _ = ioutil.TempDir("", "mt")
	}

	tmpFile, err := ioutil.TempFile(tmpDir, "mt")
	if err != nil {
		fmt.Print(err)
	}

	buf := new(bytes.Buffer)
	_ = jpeg.Encode(buf, img, nil)
	send_s3 := buf.Bytes()

	tmpFile.Write(send_s3)

	defer tmpFile.Close()

	return tmpFile.Name()
}
