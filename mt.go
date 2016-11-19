package main

import (
	"encoding/json"
	"fmt"
	"github.com/mutschler/mt/Godeps/_workspace/src/code.google.com/p/jamslam-freetype-go/freetype"
	"github.com/mutschler/mt/Godeps/_workspace/src/github.com/disintegration/gift"
	"github.com/mutschler/mt/Godeps/_workspace/src/github.com/disintegration/imaging"
	"github.com/mutschler/mt/Godeps/_workspace/src/github.com/dustin/go-humanize"
	"github.com/mutschler/mt/Godeps/_workspace/src/github.com/opennota/screengen"
	log "github.com/mutschler/mt/Godeps/_workspace/src/github.com/sirupsen/logrus"
	flag "github.com/mutschler/mt/Godeps/_workspace/src/github.com/spf13/pflag"
	"github.com/mutschler/mt/Godeps/_workspace/src/github.com/spf13/viper"
	"image"
	"image/draw"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var blankPixels int
var allPixels int
var mpath string
var fontBytes []byte
var version string = "1.0.7-dev"
var timestamps []string

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

	if viper.GetBool("webvtt") {
		timestamps = append(timestamps, "00:00:00")
	}

	for i := 0; i < viper.GetInt("numcaps"); i++ {
		stamp := d
		img, err := gen.Image(d)
		if err != nil {
			log.Fatalf("Can't generate screenshot: %v", err)
			os.Exit(1)
		}

		// should we skip any images?
		if viper.GetBool("skip_blank") || viper.GetBool("skip_blurry") || viper.GetBool("sfw") {
			maxCount := 3
			count := 1
			for skipImage(img) == true && maxCount >= count {
				log.Warnf("[%d/%d] frame skipped based on settings at: %s retry at: %s", count, maxCount, fmt.Sprintf(time.Unix(stamp/1000, 0).UTC().Format("15:04:05")), fmt.Sprintf(time.Unix((stamp+10000)/1000, 0).UTC().Format("15:04:05")))
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
		if viper.GetBool("webvtt") {
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
			if viper.GetInt("numcaps") == 1 {
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

		d += inc
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
	x := 0
	curRow := 0

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

		if viper.GetBool("webvtt") {
			_, imgName := filepath.Split(fn)
			vttContent = fmt.Sprintf("%s\n%s.000 --> %s.000\n%s#xywh=%d,%d,%d,%d\n", vttContent, timestamps[idx], timestamps[idx+1], imgName, xPos, yPos, imgWidth, imgHeight)
		}

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
	log.Infof("Saved image to %s", fn)
	if viper.GetBool("webvtt") {
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
	viper.SetDefault("skip_blurry", false)
	viper.SetDefault("skip_existing", false)
	viper.SetDefault("overwrite", false)
	viper.SetDefault("sfw", false)
	viper.SetDefault("fast", false)
	viper.SetDefault("show_config", false)
	viper.SetDefault("webvtt", false)
	viper.SetDefault("blur_threshold", 62)
	viper.SetDefault("blank_threshold", 85)
	viper.SetDefault("upload", false)
	viper.SetDefault("upload_url", "http://example.com/upload")

	flag.IntP("numcaps", "n", viper.GetInt("numcaps"), "number of captures to make")
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

	flag.BoolP("disable-timestamps", "d", viper.GetBool("disable_timestamps"), "disable timestamps on images")
	viper.BindPFlag("disable_timestamps", flag.Lookup("disable-timestamps"))

	flag.BoolP("verbose", "v", viper.GetBool("verbose"), "enable verbose output")
	viper.BindPFlag("verbose", flag.Lookup("verbose"))

	flag.BoolP("single-images", "s", viper.GetBool("single_images"), "save single images instead of one combined contact sheet")
	viper.BindPFlag("single_images", flag.Lookup("single-images"))

	flag.String("bg-header", viper.GetString("bg_header"), "rgb background color for header")
	viper.BindPFlag("bg_header", flag.Lookup("bg-header"))

	flag.String("fg-header", viper.GetString("fg_header"), "rgb font color for header")
	viper.BindPFlag("fg_header", flag.Lookup("fg-header"))

	flag.String("bg-content", viper.GetString("bg_content"), "rgb background color for the main content area")
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

	flag.Bool("skip-blurry", viper.GetBool("skip_blurry"), "skip up to 3 images in a row which seem to be blurry (can slow mt down)")
	viper.BindPFlag("skip_blurry", flag.Lookup("skip-blurry"))

	flag.Bool("version", false, "show version number and exit")
	viper.BindPFlag("show_version", flag.Lookup("version"))

	flag.Bool("header", viper.GetBool("header"), "append header to the contact sheet")
	viper.BindPFlag("header", flag.Lookup("header"))

	flag.Bool("header-meta", viper.GetBool("header_meta"), "also add codec, fps and bitrate informations to the header")
	viper.BindPFlag("header_meta", flag.Lookup("header-meta"))

	flag.String("filter", viper.GetString("filter"), "apply one or mor filters to images (comma seperated list), see --filters for available filters")
	viper.BindPFlag("filter", flag.Lookup("filter"))

	flag.Bool("filters", false, "list all available image filters")
	viper.BindPFlag("filters", flag.Lookup("filters"))

	flag.String("output", viper.GetString("filename"), "set an output filename")
	viper.BindPFlag("filename", flag.Lookup("output"))

	flag.String("from", viper.GetString("from"), "set starting point in format HH:MM:SS")
	viper.BindPFlag("from", flag.Lookup("from"))

	flag.String("to", viper.GetString("end"), "set end point in format HH:MM:SS")
	viper.BindPFlag("end", flag.Lookup("to"))

	flag.String("save-config", viper.GetString("save_config"), "save config with current settings to this file")
	viper.BindPFlag("save_config", flag.Lookup("save-config"))

	flag.String("config-file", viper.GetString("config_file"), "use a specific configuration file")
	viper.BindPFlag("config_file", flag.Lookup("config-file"))

	flag.Bool("skip-existing", viper.GetBool("skip_existing"), "skip any item if there is already a screencap present")
	viper.BindPFlag("skip_existing", flag.Lookup("skip-existing"))

	flag.Bool("overwrite", viper.GetBool("overwrite"), "overwrite existing screencaps")
	viper.BindPFlag("overwrite", flag.Lookup("overwrite"))

	flag.Bool("sfw", viper.GetBool("sfw"), "use nudity detection to generate sfw images (HIGHLY EXPERIMENTAL)")
	viper.BindPFlag("sfw", flag.Lookup("sfw"))

	flag.Bool("show-config", viper.GetBool("show_config"), "show path to currently used config file as well as used values and exit")
	viper.BindPFlag("show_config", flag.Lookup("show-config"))

	flag.Bool("fast", viper.GetBool("fast"), "inacurate but faster seeking")
	viper.BindPFlag("fast", flag.Lookup("fast"))

	flag.Bool("webvtt", viper.GetBool("webvtt"), "create a .vtt file as well")
	viper.BindPFlag("webvtt", flag.Lookup("webvtt"))

	flag.Int("blur-threshold", viper.GetInt("blur_threshold"), "set a custom threshold to use for blurry image detection (defaults to 62)")
	viper.BindPFlag("blur_threshold", flag.Lookup("blur-threshold"))

	flag.Int("blank-threshold", viper.GetInt("blank_threshold"), "set a custom threshold to use for blank image detection (defaults to 85)")
	viper.BindPFlag("blank_threshold", flag.Lookup("blank-threshold"))

	flag.Bool("upload", viper.GetBool("upload"), "post file via http form submit")
	viper.BindPFlag("upload", flag.Lookup("upload"))

	flag.String("upload-url", viper.GetString("upload_url"), "url to use for --upload")
	viper.BindPFlag("upload_url", flag.Lookup("upload-url"))

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

	if viper.GetBool("upload") {
		if viper.GetString("upload_url") == "" || viper.GetString("upload_url") == "http://example.com/upload" {
			log.Errorf("can't use upload function without an url! please use --upload-url");
		}
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

	if len(flag.Args()) == 0 && !viper.GetBool("show_config") {
		flag.Usage()
		os.Exit(1)
	}

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	if viper.GetBool("webvtt") {
		viper.Set("header", false)
		viper.Set("header_meta", false)
		viper.Set("padding", 0)
	}

	// print config file and used values!
	if viper.ConfigFileUsed() != "" {
		log.Debugf("Config file used: %s", viper.ConfigFileUsed())
	} else {
		log.Debugf("Useing default and runtime config")
	}

	b, _ := json.Marshal(viper.AllSettings())
	log.Debugf("config values: %s", b)

	fontBytes, err = getFont(viper.GetString("font_all"))
	if err != nil {
		log.Warn("unable to load font, disableing timestamps and header")
	}

	if viper.GetBool("show_config") {
		if viper.ConfigFileUsed() != "" {
			log.Infof("Config file used: %s", viper.ConfigFileUsed())
		} else {
			log.Infof("Useing default and runtime config")
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
