package main

import (
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/code.google.com/p/jamslam-freetype-go/freetype"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/disintegration/gift"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/disintegration/imaging"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/dustin/go-humanize"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/koyachi/go-nude"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/opennota/screengen"
	log "bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/sirupsen/logrus"
	flag "bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/spf13/pflag"
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/spf13/viper"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var blankPixels int
var allPixels int
var mpath string
var fontBytes []byte
var version string = "1.0.5-dev"

func randomInt(min, max int) float32 {
	rand.Seed(time.Now().UTC().UnixNano())
	return float32(rand.Intn(max-min) + min)
}

func fileExists(fname string) bool {
	if _, err := os.Stat(fname); err == nil {
		return true
	}
	return false
}

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

//gets the timestamp value ("HH:MM:SS") and returns an image
//TODO: rework this to take any string and a bool for full width/centered text
func drawTimestamp(timestamp string) image.Image {
	var timestamped image.Image

	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Error(err)
		return timestamped
	}

	fg, bg := image.White, image.Black
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetFontSize(float64(viper.GetInt("font_size")))

	// get width and height of the string and draw an image to hold it
	x, y, _ := c.MeasureString(timestamp)
	rgba := image.NewRGBA(image.Rect(0, 0, (int(x)/256)+10, (int(y)/256)+10))
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)

	//draw the text with 5px padding
	pt := freetype.Pt(5, 3+int(c.PointToFix32(float64(viper.GetInt("font_size")))>>8))
	_, err = c.DrawString(timestamp, pt)
	if err != nil {
		log.Errorf("error creating timestamp image for: %s", timestamp)
		return timestamped
	}

	log.Debugf("created timestamp image for: %s", timestamp)

	return rgba

}

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

// generates screenshots and returns a list of images
func GenerateScreenshots(fn string) []image.Image {
	var thumbnails []image.Image
	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		log.Fatalf("Error reading video file: %v", err)
		os.Exit(1)
	}

	if viper.GetBool("fast") {
		gen.Fast = true
	}

	defer gen.Close()

	duration := gen.Duration
	from := int64(0)
	end := int64(0)

	if viper.GetString("from") != "0" {
		log.Infof("First screenshot will be at %s", viper.GetString("from"))
		from = stringToMS(viper.GetString("from"))
	}
	if viper.GetString("end") != "0" {
		log.Infof("Last screenshot will be at %s", viper.GetString("end"))
		end = stringToMS(viper.GetString("end"))
	}

	percentage := int64((float32(duration / 100)) * (5.5 * 2))
	//cut of 2 minutes of video if video has at least 4 minutes else cut away (or at least 10.10%)
	if duration > (120000*2) && 120000 > percentage {
		duration = duration - 120000
	} else {
		duration = duration - percentage
	}

	if end > 0 {
		duration = end
	}

	if from > 0 {
		duration = duration - from
	}

	inc := duration / (int64(viper.GetInt("numcaps")))

	if end > 0 && from > 0 {
		inc = duration / (int64(viper.GetInt("numcaps")) - 1)
	}

	if inc <= 60000 {
		log.Warn("very small timestamps in use... consider decreasing numcaps")
	}
	if inc <= 9000 {
		log.Errorf("interval (%ds) is way to small (less then 9s), please decrease numcaps", inc/1000)
	}

	d := inc
	if from > 0 {
		d = from
	}

	for i := 0; i < viper.GetInt("numcaps"); i++ {
		stamp := d
		img, err := gen.Image(d)
		if err != nil {
			log.Fatalf("Can't generate screenshot: %v", err)
			os.Exit(1)
		}
		//try to detect images with a large black/white amount
		if viper.GetBool("skip_blank") {
			maxCount := 3
			count := 1
			for isBlankImage(img) == true && maxCount >= count {
				log.Warnf("[%d/%d] blank frame detected at: %s retry at: %s", count, maxCount, fmt.Sprintf(time.Unix(stamp/1000, 0).UTC().Format("15:04:05")), fmt.Sprintf(time.Unix((stamp+10000)/1000, 0).UTC().Format("15:04:05")))
				if stamp >= duration-inc {
					log.Error("end of clip reached... no more blank frames can be skipped")
					i = viper.GetInt("numcaps") - 1
					break
				}
				stamp = d + (10000 * int64(count))
				img, _ = gen.Image(stamp)
				count = count + 1
			}
		}

		if viper.GetBool("sfw") {
			maxCount := 3
			count := 1
			for isNudeImage(img) == true && maxCount >= count {
				log.Warnf("[%d/%d] nude image detected at: %s retry at: %s", count, maxCount, fmt.Sprintf(time.Unix(stamp/1000, 0).UTC().Format("15:04:05")), fmt.Sprintf(time.Unix((stamp+10000)/1000, 0).UTC().Format("15:04:05")))
				if stamp >= duration-inc {
					log.Error("end of clip reached... no more blank frames can be skipped")
					i = viper.GetInt("numcaps") - 1
					break
				}
				stamp = d + (10000 * int64(count))
				img, _ = gen.Image(stamp)
				count = count + 1
			}
		}

		// if we skipped ahead of next frame...
		if stamp > d {
			d = stamp
		}

		timestamp := fmt.Sprintf(time.Unix(stamp/1000, 0).UTC().Format("15:04:05"))
		log.Infof("generating screenshot %02d/%02d at %s", i+1, viper.GetInt("numcaps"), timestamp)
		//var thumb image.Image
		if viper.GetInt("width") > 0 {
			img = imaging.Resize(img, viper.GetInt("width"), 0, imaging.Lanczos)
		} else if viper.GetInt("width") == 0 && viper.GetInt("height") > 0 {
			img = imaging.Resize(img, 0, viper.GetInt("height"), imaging.Lanczos)
		}

		//apply filters
		filters := strings.Split(viper.GetString("filter"), ",")
		for _, filter := range filters {
			switch filter {
			case "greyscale":
				img = imaging.Grayscale(img)
				img = imaging.Sharpen(img, 1.0)
				img = imaging.AdjustContrast(img, 20)
				log.Debug("greyscale filter applied")
			case "invert":
				img = imaging.Invert(img)
				log.Debug("invert filter applied")
			case "fancy":
				//TODO: find a way to do this without GIFT...
				log.Debug("fancy filter applied")
				//draw timestamp to the image before rotating it!
				tsimage := drawTimestamp(timestamp)
				img = imaging.Overlay(img, tsimage, image.Pt(img.Bounds().Dx()-tsimage.Bounds().Dx()-10, img.Bounds().Dy()-tsimage.Bounds().Dy()-10), viper.GetFloat64("timestamp_opacity"))

				g := gift.New(
					gift.Rotate(randomInt(-10, 15), getImageColor(viper.GetString("bg_content"), []int{0, 0, 0}), gift.CubicInterpolation),
				)
				dst := image.NewRGBA(g.Bounds(img.Bounds()))
				g.Draw(dst, img)
				img = dst
				viper.Set("disable_timestamps", true)
			case "sepia":
				log.Debug("sepia filter applied")
				g := gift.New(
					gift.Sepia(100),
				)
				dst := image.NewRGBA(g.Bounds(img.Bounds()))
				g.Draw(dst, img)
				img = dst
			case "cross":
				log.Debug("cross filter applied")
				img = CrossProcessingFilter(img, 0.5, 9)
			case "strip":
				log.Debug("image stip filter applied")
				//draw timestamp!
				tsimage := drawTimestamp(timestamp)
				img = imaging.Overlay(img, tsimage, image.Pt(img.Bounds().Dx()-tsimage.Bounds().Dx()-10, img.Bounds().Dy()-tsimage.Bounds().Dy()-10), viper.GetFloat64("timestamp_opacity"))
				viper.Set("disable_timestamps", true)
				img = ImageStripFilter(img)
			}
		}

		if !viper.GetBool("disable_timestamps") && !viper.GetBool("single_images") {
			log.Debug("adding timestamp to image")
			tsimage := drawTimestamp(timestamp)
			img = imaging.Overlay(img, tsimage, image.Pt(img.Bounds().Dx()-tsimage.Bounds().Dx()-10, img.Bounds().Dy()-tsimage.Bounds().Dy()-10), viper.GetFloat64("timestamp_opacity"))
		}

		//watermark middle image
		if i == (viper.GetInt("numcaps")-1)/2 && viper.GetString("watermark") != "" && !viper.GetBool("single_images") {
			ov, err := imaging.Open(viper.GetString("watermark"))
			if err == nil {
				if ov.Bounds().Dx() > img.Bounds().Dx() {
					ov = imaging.Resize(ov, img.Bounds().Dx(), 0, imaging.Lanczos)
				}
				if ov.Bounds().Dy() > img.Bounds().Dy() {
					ov = imaging.Resize(ov, 0, img.Bounds().Dy(), imaging.Lanczos)
				}
				posX := (img.Bounds().Dx() - ov.Bounds().Dx()) / 2
				posY := (img.Bounds().Dy() - ov.Bounds().Dy()) / 2
				img = imaging.Overlay(img, ov, image.Pt(posX, posY), 0.6)
			}
		} else if viper.GetString("watermark") != "" && viper.GetBool("single_images") {
			ov, err := imaging.Open(viper.GetString("watermark"))
			if err == nil {
				if ov.Bounds().Dx() > img.Bounds().Dx() {
					ov = imaging.Resize(ov, img.Bounds().Dx(), 0, imaging.Lanczos)
				}
				if ov.Bounds().Dy() > img.Bounds().Dy() {
					ov = imaging.Resize(ov, 0, img.Bounds().Dy(), imaging.Lanczos)
				}
				posX := (img.Bounds().Dx() - ov.Bounds().Dx()) / 2
				posY := (img.Bounds().Dy() - ov.Bounds().Dy()) / 2
				img = imaging.Overlay(img, ov, image.Pt(posX, posY), 0.6)
			}
		}

		if viper.GetString("watermark_all") != "" {
			ov, err := imaging.Open(viper.GetString("watermark_all"))
			if err == nil {
				if ov.Bounds().Dx() > (img.Bounds().Dx() / 4) {
					ov = imaging.Resize(ov, (img.Bounds().Dx()/4), 0, imaging.Lanczos)
				}
				if ov.Bounds().Dy() > (img.Bounds().Dy()/4) {
					ov = imaging.Resize(ov, 0, (img.Bounds().Dy()/4), imaging.Lanczos)
				}
				//default position for watermarking is bottom-left
				posX := 10
				posY := img.Bounds().Dy() - ov.Bounds().Dy() - 10
				img = imaging.Overlay(img, ov, image.Pt(posX, posY), 0.6)
			}
		}


		if viper.GetBool("single_images") {
			fname := getSavePath(mpath, i+1)
			createTargetDirs(fname)
			imaging.Save(img, fname)
		} else {
			thumbnails = append(thumbnails, img)
		}

		d += inc
	}

	return thumbnails
}

func createTargetDirs(fn string) {
	path, _ := filepath.Split(fn)
	os.MkdirAll(path, 0777)
}

func makeContactSheet(thumbs []image.Image, fn string) {
	log.Info("Composing Contact Sheet")
	imgWidth := thumbs[0].Bounds().Dx()
	imgHeight := thumbs[0].Bounds().Dy()

	columns := viper.GetInt("columns")
	imgRows := int(math.Ceil(float64(len(thumbs)) / float64(columns)))

	log.Debugf("single image dimension: %dx%d", imgWidth, imgHeight)
	log.Debugf("new image dimension: %dx%d", imgWidth*columns, imgHeight*imgRows)

	paddingColumns := 0
	singlepadd := 0
	paddingRows := 0
	if viper.GetInt("padding") > 0 {
		paddingColumns = (columns + 1) * viper.GetInt("padding")
		paddingRows = (imgRows + 1) * viper.GetInt("padding")
		singlepadd = viper.GetInt("padding")
	}

	// create a new blank image
	bgColor := getImageColor(viper.GetString("bg_content"), []int{0, 0, 0})
	dst := imaging.New(imgWidth*columns+paddingColumns, imgHeight*imgRows+paddingRows, bgColor)
	x := 0
	curRow := 0
	// paste thumbnails into the new image side by side with padding if enabled
	for _, thumb := range thumbs {

		if x >= columns {
			x = 0
			curRow = curRow + 1
		}

		xPos := (x * imgWidth) + singlepadd
		yPos := (curRow * imgHeight) + singlepadd

		if x >= 0 && x <= columns {
			xPos = (x * imgWidth) + (singlepadd * x) + singlepadd
		}

		if curRow >= 0 && curRow < imgRows {
			yPos = (curRow * imgHeight) + (singlepadd * curRow) + singlepadd
		}

		dst = imaging.Paste(dst, thumb, image.Pt(xPos, yPos))
		x = x + 1
	}

	if viper.GetBool("header") {
		log.Info("appending header informations")
		head := appendHeader(dst)
		newIm := imaging.New(dst.Bounds().Dx(), dst.Bounds().Dy()+head.Bounds().Dy(), bgColor)
		dst = imaging.Paste(newIm, dst, image.Pt(0, head.Bounds().Dy()))
		dst = imaging.Paste(dst, head, image.Pt(0, 0))
	}

	// save the combined image to file
	createTargetDirs(fn)
	err := imaging.Save(dst, fn)
	if err != nil {
		log.Fatalf("error saveing image: %v", err)
	}
	log.Infof("Saved to %s", fn)
}

func appendHeader(im image.Image) image.Image {
	var timestamped image.Image

	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Errorf("freetype parse error", err)
		return timestamped
	}

	// TODO: move this to a helper function!
	bgColor := getImageColor(viper.GetString("bg_header"), []int{0, 0, 0})

	fgColor := getImageColor(viper.GetString("fg_header"), []int{255, 255, 255})

	fontcolor, bg := image.NewUniform(fgColor), image.NewUniform(bgColor)
	c := freetype.NewContext()
	c.SetDPI(96)
	c.SetFont(font)
	c.SetFontSize(float64(viper.GetInt("font_size")))

	// get width and height of the string and draw an image to hold it
	//x, y, _ := c.MeasureString(timestamp)
	header := createHeader(mpath)

	rgba := image.NewNRGBA(image.Rect(0, 0, im.Bounds().Dx(), (5+int(c.PointToFix32(float64(viper.GetInt("font_size")+4))>>8)*len(header))+10))
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	if viper.GetString("header_image") != "" {
		ov, err := imaging.Open(viper.GetString("header_image"))
		if err == nil {
			if ov.Bounds().Dy() >= (rgba.Bounds().Dy() - 20) {
				ov = imaging.Resize(ov, 0, rgba.Bounds().Dy()-20, imaging.Lanczos)
			}
			//center image inside header
			posY := (rgba.Bounds().Dy() - ov.Bounds().Dy()) / 2
			if posY < 10 {
				posY = 10
			}
			rgba = imaging.Overlay(rgba, ov, image.Pt(rgba.Bounds().Dx()-ov.Bounds().Dx()-10, posY), 1.0)

		} else {
			log.Error("error opening header overlay image")
		}
	}

	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fontcolor)

	for i, s := range header {
		//draw the text with 10px padding and lineheight +4
		pt := freetype.Pt(10, (5 + int(c.PointToFix32(float64(viper.GetInt("font_size")+4))>>8)*(i+1)))
		_, err = c.DrawString(s, pt)
		if err != nil {
			fmt.Println(err)
			return timestamped
		}
	}

	return rgba
}

func createHeader(fn string) []string {

	var header []string
	_, fname := filepath.Split(fn)

	f, err := os.Open(fn)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		fmt.Println(err)
	}
	fsize := fmt.Sprintf("File Size: %s", humanize.Bytes(uint64(stat.Size())))
	fname = fmt.Sprintf("File Name: %s", fname)

	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		log.Errorf("Error reading video file: %v", err)
		os.Exit(1)
	}
	defer gen.Close()

	duration := fmt.Sprintf("Duration: %s", time.Unix(gen.Duration/1000, 0).UTC().Format("15:04:05"))

	dimension := fmt.Sprintf("Resolution: %dx%d", gen.Width, gen.Height)

	header = append(header, fname)
	header = append(header, fsize)
	header = append(header, duration)
	header = append(header, dimension)

	if viper.GetBool("header_meta") {
		header = append(header, fmt.Sprintf("FPS: %.2f, Bitrate: %dKbp/s", gen.FPS, gen.Bitrate))
		header = append(header, fmt.Sprintf("Codec: %s / %s", gen.VideoCodecLongName, gen.AudioCodecLongName))
	}

	if viper.GetString("comment") != "" {
		header = append(header, fmt.Sprintf("%s", viper.GetString("comment")))
	}

	return header
}

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

func stringToMS(s string) int64 {
	x := strings.Split(s, ":")
	hh, _ := strconv.Atoi(x[0])
	mm, _ := strconv.Atoi(x[1])
	ss, _ := strconv.Atoi(x[2])

	end := (ss + (mm * 60) + (hh * 60 * 60)) * 1000
	return int64(end)
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	viper.SetConfigName("mt")
	viper.SetEnvPrefix("mt")
	viper.SetDefault("numcaps", 4)
	viper.SetDefault("columns", 2)
	viper.SetDefault("padding", 10)
	viper.SetDefault("width", 400)
	viper.SetDefault("height", 0)
	viper.SetDefault("font_all", "DroidSans.ttf")
	viper.SetDefault("font_size", 12)
	viper.SetDefault("disable_timestamps", false)
	viper.SetDefault("timestamp_opacity", 1.0)
	viper.SetDefault("filename", "{{.Path}}{{.Name}}.jpg")
	viper.SetDefault("verbose", false)
	viper.SetDefault("bg_content", "0,0,0")
	viper.SetDefault("border", 0)
	viper.SetDefault("from", "0")
	viper.SetDefault("end", "0")
	viper.SetDefault("single_images", false)
	viper.SetDefault("header", true)
	viper.SetDefault("font_dirs", []string{})
	viper.SetDefault("bg_header", "0,0,0")
	viper.SetDefault("fg_header", "255,255,255")
	viper.SetDefault("header_image", "")
	viper.SetDefault("header_meta", false)
	viper.SetDefault("watermark", "")
	viper.SetDefault("comment", "")
	viper.SetDefault("watermark-all", "")
	viper.SetDefault("filter", "none")
	viper.SetDefault("skip_blank", false)
	viper.SetDefault("skip_existing", false)
	viper.SetDefault("overwrite", false)
	viper.SetDefault("sfw", false)
	viper.SetDefault("fast", false)

	flag.IntP("numcaps", "n", viper.GetInt("numcaps"), "number of captures")
	viper.BindPFlag("numcaps", flag.Lookup("numcaps"))

	flag.IntP("columns", "c", viper.GetInt("columns"), "number of columns")
	viper.BindPFlag("columns", flag.Lookup("columns"))

	flag.IntP("padding", "p", viper.GetInt("padding"), "padding between the images in px")
	viper.BindPFlag("padding", flag.Lookup("padding"))

	flag.IntP("width", "w", viper.GetInt("width"), "width of a single screenshot in px")
	viper.BindPFlag("width", flag.Lookup("width"))

	flag.StringP("font", "f", viper.GetString("font_all"), "font to use for timestamps and header information")
	viper.BindPFlag("font_all", flag.Lookup("font_all"))

	flag.Int("font-size", viper.GetInt("font_size"), "font size in px")
	viper.BindPFlag("font_size", flag.Lookup("font-size"))

	flag.BoolP("disable-timestamps", "d", viper.GetBool("disable_timestamps"), "disable-timestamps")
	viper.BindPFlag("disable_timestamps", flag.Lookup("disable-timestamps"))

	flag.BoolP("verbose", "v", viper.GetBool("verbose"), "verbose output")
	viper.BindPFlag("verbose", flag.Lookup("verbose"))

	flag.BoolP("single-images", "s", viper.GetBool("single_images"), "save single images instead of one combined contact sheet")
	viper.BindPFlag("single_images", flag.Lookup("single-images"))

	flag.String("bg-header", viper.GetString("bg_header"), "rgb background color for header")
	viper.BindPFlag("bg_header", flag.Lookup("bg-header"))

	flag.String("fg-header", viper.GetString("fg_header"), "rgb font color for header")
	viper.BindPFlag("fg_header", flag.Lookup("fg-header"))

	flag.String("bg-content", viper.GetString("bg_content"), "rgb background color for header")
	viper.BindPFlag("bg_content", flag.Lookup("bg-content"))

	flag.String("header-image", viper.GetString("header_image"), "image to put in the header")
	viper.BindPFlag("header_image", flag.Lookup("header-image"))

	flag.String("comment", viper.GetString("comment"), "Add a text comment to the header")
	viper.BindPFlag("comment", flag.Lookup("comment"))

	flag.String("watermark", viper.GetString("watermark"), "watermark the final image")
	viper.BindPFlag("watermark", flag.Lookup("watermark"))

	flag.String("watermark-all", viper.GetString("watermark_all"), "watermark every single image")
	viper.BindPFlag("watermark_all", flag.Lookup("watermark-all"))

	flag.BoolP("skip-blank", "b", viper.GetBool("skip_blank"), "skip up to 3 images in a row which seem to be blank (can slow mt down)")
	viper.BindPFlag("skip_blank", flag.Lookup("skip-blank"))

	flag.Bool("version", false, "show version number and exit")
	viper.BindPFlag("show_version", flag.Lookup("version"))

	flag.Bool("header", viper.GetBool("header"), "append header to the contact sheet")
	viper.BindPFlag("header", flag.Lookup("header"))

	flag.Bool("header-meta", viper.GetBool("header_meta"), "append codec, fps and bitrate informations to the header")
	viper.BindPFlag("header_meta", flag.Lookup("header-meta"))

	flag.String("filter", viper.GetString("filter"), "apply filter to images, see --filters for available filters")
	viper.BindPFlag("filter", flag.Lookup("filter"))

	flag.Bool("filters", false, "list all available filters")
	viper.BindPFlag("filters", flag.Lookup("filters"))

	flag.String("output", viper.GetString("filename"), "set an output filename")
	viper.BindPFlag("filename", flag.Lookup("output"))

	flag.String("from", viper.GetString("from"), "set starting point in format HH:MM:SS")
	viper.BindPFlag("from", flag.Lookup("from"))

	flag.String("to", viper.GetString("end"), "set end point in format HH:MM:SS")
	viper.BindPFlag("end", flag.Lookup("to"))

	flag.String("save-config", viper.GetString("save_config"), "save config with current settings to this path")
	viper.BindPFlag("save_config", flag.Lookup("save-config"))

	flag.String("config-file", viper.GetString("config_file"), "use this configuration file")
	viper.BindPFlag("config_file", flag.Lookup("config-file"))

	flag.Bool("skip-existing", viper.GetBool("skip_existing"), "skip any item if there is already a screencap present")
	viper.BindPFlag("skip_existing", flag.Lookup("skip-existing"))

	flag.Bool("overwrite", viper.GetBool("overwrite"), "overwrite existing screencaps")
	viper.BindPFlag("overwrite", flag.Lookup("overwrite"))

	flag.Bool("sfw", viper.GetBool("sfw"), "use nudity detection to generate sfw images (HIGHLY EXPERIMENTAL)")
	viper.BindPFlag("sfw", flag.Lookup("sfw"))

	flag.Bool("fast", viper.GetBool("fast"), "inacurate but faster seeking")
	viper.BindPFlag("fast", flag.Lookup("fast"))

	viper.AutomaticEnv()

	viper.SetConfigType("json")
	viper.AddConfigPath("/etc/mt/")
	viper.AddConfigPath("$HOME/.mt")
	viper.AddConfigPath("./")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags] [file]\n", os.Args[0])
		flag.PrintDefaults()
	}

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Errorf("error reading config file: %s useing default values", err)
	}

	flag.Parse()

	if viper.GetBool("show_version") {
		fmt.Fprintf(os.Stderr, "mt Version %s\n", version)
		os.Exit(1)
	}

	if viper.GetString("config_file") != "" {
		log.Debugf("Useing custom config file stored at: %s", viper.GetString("config_file"))
		viper.SetConfigFile(viper.GetString("config_file"))
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Errorf("error reading config file: %s useing default values", err)
		}
	}

	if viper.GetString("save_config") != "" {
		saveConfig(viper.GetString("save_config"))
	}

	if viper.GetBool("filters") {
		allFilters := `available image filters:

| NAME      | DESCRIPTION                     |
| --------- | --------------------------------|
| invert    | invert colors                   |
| greyscale | convert to greyscale image      |
| sepia     | convert to sepia image          |
| fancy     | randomly rotates every image    |
| cross     | simulated cross processing      |
| strip     | simulate an old 35mm Film strip |

you can stack multiple filters by seperating them with a comma
example:

    --filter=cross,fancy 

NOTE: fancy has best results if it is applied as last filter!

`
		fmt.Fprintf(os.Stderr, allFilters)
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	// print config file and used values!
	log.Debugf("Config file used: %s", viper.ConfigFileUsed())
	b, _ := json.Marshal(viper.AllSettings())
	log.Debugf("config values: %s", b)

	fontBytes, err = getFont(viper.GetString("font_all"))
	if err != nil {
		log.Warn("unable to load font, disableing timestamps and header")
	}

	for _, movie := range flag.Args() {
		mpath = movie
		log.Infof("generating contact sheet for %s", movie)
		log.Debugf("image will be saved as %s", getSavePath(movie, 0))

		var thumbs []image.Image
		// thumbs = getImages(movie)
		// TODO: implement generation of image contac sheets from a folder

		//skip existing image if option is present
		if fileExists(getSavePath(movie, 0)) && viper.GetBool("skip_existing") {
			log.Infof("file already exists, skipping %s", getSavePath(movie, 0))
			continue
		}

		thumbs = GenerateScreenshots(movie)
		if len(thumbs) > 0 {
			makeContactSheet(thumbs, getSavePath(movie, 0))
		}

	}

}
