package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"io/ioutil"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/freetype-go/freetype"
	"github.com/disintegration/gift"
	"github.com/disintegration/imaging"
	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gitlab.com/opennota/screengen"
)

var GitVersion = ""
var FfmpegVersion = ""
var BuildTimestamp = ""

var blankPixels int
var allPixels int
var mpath string
var fontBytes []byte
var version string = GitVersion + " (" + FfmpegVersion + ") built on " + BuildTimestamp
var timestamps []string
var numcaps int

// gets the timestamp value ("HH:MM:SS") and returns an image
// TODO: rework this to take any string and a bool for full width/centered text
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

// generates screenshots and returns a list of images
func GenerateScreenshots(fn string) []image.Image {
	var thumbnails []image.Image
	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		log.Fatalf("Error reading video file: %v", err)
	}

	if viper.GetBool("fast") {
		gen.Fast = true
	}

	defer gen.Close()

	// truncate duration to full seconds
	// this prevents empty/black images when the movie is some milliseconds longer
	// ffmpeg then sometimes takes a black screenshot AFTER the movie finished for some reason
	duration := 1000 * (gen.Duration / 1000)
	from := stringToMS(viper.GetString("from"))
	end := stringToMS(viper.GetString("end"))

	if from > end {
		log.Fatalf("from cant be higher than to")
	}
	if from > 0 {
		log.Infof("First screenshot will be at %s", viper.GetString("from"))
	}
	if end > 0 && from < end {
		log.Infof("Last screenshot will be at %s", viper.GetString("end"))
	}

	if viper.GetBool("skip_credits") {
		percentage := int64((float32(duration / 100)) * (5.5 * 2))
		//cut of 2 minutes of video if video has at least 4 minutes else cut away (or at least 10.10%)
		if duration > (120000*2) && 120000 > percentage {
			duration = duration - 120000
		} else {
			duration = duration - percentage
		}
	}

	if end > 0 {
		duration = end
	}

	if from > 0 {
		duration = duration - from
	}

	numcaps = viper.GetInt("numcaps")
	if viper.GetInt("interval") > 0 {
		var durationSec = duration / 1000
		var intervalSec = int64(viper.GetInt("interval"))
		if durationSec < intervalSec {
			log.Fatalf("Specified interval is longer than video duration, " +
				"use smaller interval or set numcaps instead.")
		}
		numcaps = int(durationSec / intervalSec)
		log.Debugf("interval option set, numcaps are set to %d", numcaps)
		viper.Set("columns", int(math.Sqrt(float64(numcaps))))
	}

	inc := duration / (int64(numcaps))

	if end > 0 && from > 0 && numcaps > 1 {
		inc = duration / (int64(numcaps) - 1)
	}

	if inc <= 60000 {
		log.Warn("very small timestamps in use... consider decreasing numcaps")
	}
	if inc <= 9000 {
		log.Errorf("interval (%ds) is way to small (less then 9s), please decrease numcaps", inc/1000)
	}

	d := inc

	if viper.GetInt("interval") > 0 {
		d = (int64(viper.GetInt("interval")) * 1000)
	}

	if from > 0 {
		d = from
	}

	if viper.GetBool("vtt") {
		timestamps = append(timestamps, "00:00:00")
	}

	for i := 0; i < numcaps; i++ {
		stamp := d
		img, err := gen.Image(d)
		if err != nil {
			log.Fatalf("Can't generate screenshot: %v", err)
		}

		// should we skip any images?
		if viper.GetBool("skip_blank") || viper.GetBool("skip_blurry") || viper.GetBool("sfw") {
			maxCount := 3
			count := 1
			for skipImage(img) == true && maxCount >= count {
				log.Warnf("[%d/%d] frame skipped based on settings at: %s retry at: %s", count, maxCount, fmt.Sprintf(time.Unix(stamp/1000, 0).UTC().Format("15:04:05")), fmt.Sprintf(time.Unix((stamp+10000)/1000, 0).UTC().Format("15:04:05")))
				if stamp >= duration-inc {
					log.Error("end of clip reached... no more blank frames can be skipped")
					i = numcaps - 1
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
		log.Infof("generating screenshot %02d/%02d at %s", i+1, numcaps, timestamp)
		if viper.GetBool("vtt") {
			timestamps = append(timestamps, timestamp)
		}
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
		if i == (numcaps-1)/2 && viper.GetString("watermark") != "" && !viper.GetBool("single_images") {
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
					ov = imaging.Resize(ov, (img.Bounds().Dx() / 4), 0, imaging.Lanczos)
				}
				if ov.Bounds().Dy() > (img.Bounds().Dy() / 4) {
					ov = imaging.Resize(ov, 0, (img.Bounds().Dy() / 4), imaging.Lanczos)
				}
				//default position for watermarking is bottom-left
				posX := 10
				posY := img.Bounds().Dy() - ov.Bounds().Dy() - 10
				img = imaging.Overlay(img, ov, image.Pt(posX, posY), 0.6)
			}
		}

		if viper.GetBool("single_images") {
			var fname string
			if numcaps == 1 {
				fname = getSavePath(mpath, 0)
			} else {
				fname = getSavePath(mpath, i+1)
			}
			createTargetDirs(fname)
			imaging.Save(img, fname)

			uploadFile(fname)

		} else {
			thumbnails = append(thumbnails, img)
		}

		if viper.GetInt("interval") > 0 {
			d += (int64(viper.GetInt("interval")) * 1000)
		} else {
			d += inc
		}
	}

	return thumbnails
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
	var head image.Image
	x := 0
	curRow := 0
	headerHeight := 0

	if viper.GetBool("header") {
		log.Info("creating header information")
		head = appendHeader(dst)
		headerHeight = head.Bounds().Dy()
	}

	vttContent := "WEBVTT\n"
	// paste thumbnails into the new image side by side with padding if enabled
	for idx, thumb := range thumbs {

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

		if viper.GetBool("vtt") {
			_, imgName := filepath.Split(fn)
			vttContent = fmt.Sprintf("%s\n%s.000 --> %s.000\n%s#xywh=%d,%d,%d,%d\n", vttContent, timestamps[idx], timestamps[idx+1], imgName, xPos, yPos+headerHeight, imgWidth, imgHeight)
		}

	}

	if viper.GetBool("header") {
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
	log.Infof("Saved image to %s", fn)
	if viper.GetBool("vtt") {
		vttfn := strings.Replace(fn, filepath.Ext(fn), ".vtt", -1)
		err = ioutil.WriteFile(vttfn, []byte(vttContent), 0644)
		if err != nil {
			log.Fatalf("error saveing vtt file: %v", err)
		}
		log.Infof("Saved vtt to %s", vttfn)
	}

	uploadFile(fn)

}

func appendHeader(im image.Image) image.Image {
	var timestamped image.Image

	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Errorf("freetype parse error: %v", err)
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
	var fname, fsize string
	_, fname = filepath.Split(fn)

	if f, err := os.Open(fn); err == nil {
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			fmt.Println(err)
		}
		fsize = humanize.IBytes(uint64(stat.Size()))
	} else {
		// try if it is a web video
		if _, err = url.ParseRequestURI(fn); err != nil {
			fmt.Println(err)
		}
		resp, err := http.Head(fn)
		if err != nil {
			fmt.Println(err)
		}
		defer resp.Body.Close()

		fsize = resp.Header.Get("Content-Length")
		if fsize == "" {
			fsize = "unknown" // too expensive to download the whole file
		} else {
			size, _ := strconv.Atoi(fsize)
			fsize = humanize.IBytes(uint64(size))
		}

		cdisposition := resp.Header.Get("Content-Disposition")
		_, params, err := mime.ParseMediaType(cdisposition)
		if params["filename"] != "" {
			fname = params["filename"] // prefer filename to the name split from url
		}
	}
	fsize = fmt.Sprintf("File Size: %s", fsize)
	fname = fmt.Sprintf("File Name: %s", fname)

	gen, err := screengen.NewGenerator(fn)
	if err != nil {
		log.Errorf("Error reading video file: %v", err)
		os.Exit(1)
	}
	defer gen.Close()

	duration := fmt.Sprintf("Duration: %s", time.Unix(gen.Duration/1000, 0).UTC().Format("15:04:05"))

	dimension := fmt.Sprintf("Resolution: %dx%d", gen.Width(), gen.Height())

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

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	configInit()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [flags] [file]\n", os.Args[0])
		flag.PrintDefaults()
	}

	if viper.GetBool("show_version") {
		fmt.Fprintf(os.Stderr, "mt Version %s\n", version)
		os.Exit(1)
	}

	if viper.GetBool("upload") {
		if viper.GetString("upload_url") == "" || viper.GetString("upload_url") == "http://example.com/upload" {
			log.Errorf("can't use upload function without an url! please use --upload-url")
		}
	}

	if viper.GetString("config_file") != "" {
		log.Debugf("Using custom config file stored at: %s", viper.GetString("config_file"))
		viper.SetConfigFile(viper.GetString("config_file"))
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Errorf("error reading config file: %s using default values", err)
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

	if len(flag.Args()) == 0 && !viper.GetBool("show_config") {
		flag.Usage()
		os.Exit(1)
	}

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if viper.GetBool("webvtt") {
		viper.Set("vtt", true)
		viper.Set("header", false)
		viper.Set("header_meta", false)
		viper.Set("disable_timestamps", true)
		viper.Set("padding", 0)
	}

	// print config file and used values!
	if viper.ConfigFileUsed() != "" {
		log.Debugf("Config file used: %s", viper.ConfigFileUsed())
	} else {
		log.Debugf("Using default and runtime config")
	}

	b, _ := json.Marshal(viper.AllSettings())
	log.Debugf("config values: %s", b)

	var errFont error
	fontBytes, errFont = getFont(viper.GetString("font_all"))
	if errFont != nil {
		log.Warn("unable to load font, disabling timestamps and header")
	}

	if viper.GetBool("show_config") {
		if viper.ConfigFileUsed() != "" {
			log.Infof("Config file used: %s", viper.ConfigFileUsed())
		} else {
			log.Infof("Using default and runtime config")
		}
		log.Infof("config values: %s", b)
		os.Exit(1)
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
