package main

import (
    "code.google.com/p/jamslam-freetype-go/freetype"
    "code.google.com/p/jamslam-freetype-go/freetype/truetype"
    "fmt"
    log "github.com/Sirupsen/logrus"
    "github.com/cytec/screengen"
    "github.com/disintegration/imaging"
    "github.com/dustin/go-humanize"
    flag "github.com/spf13/pflag"
    "github.com/spf13/viper"
    "image"
    "image/color"
    "image/draw"
    "image/jpeg"
    "io/ioutil"
    "math"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

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

func writeImage(img image.Image, fn string) {
    f, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatalf("Can't create file: %v", err)
        os.Exit(1)
    }
    defer f.Close()

    err = jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
    if err != nil {
        log.Fatalf("JPEG encoding error: %v", err)
        os.Exit(1)
    }
}

func Width(s string, f *truetype.Font) int {
    // scale converts truetype.FUnit to float64
    scale := float64(viper.GetInt("font_size")) / float64(f.FUnitsPerEm())

    width := 0
    prev, hasPrev := truetype.Index(0), false
    for _, rune := range s {
        index := f.Index(rune)
        if hasPrev {
            width += int(f.Kerning(f.FUnitsPerEm(), prev, index))
        }
        width += int(f.HMetric(f.FUnitsPerEm(), index).AdvanceWidth)
        prev, hasPrev = index, true
    }
    return int(float64(width)*scale) + 10
}

//gets the timestamp value ("HH:MM:SS") and returns an image
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
    defer gen.Close()

    duration := gen.Duration
    percentage := int64((float32(gen.Duration / 100)) * (5.5 * 2))
    //cut of 2 minutes of video if video has at least 4 minutes else cut away (or at least 10.10%)
    if duration > (120000*2) && 120000 > percentage {
        duration = duration - 120000
    } else {
        duration = gen.Duration - percentage
    }

    inc := duration / (int64(viper.GetInt("numcaps")))
    if inc <= 60000 {
        log.Warn("verry small timestamps in use... consider decreasing numcaps")
    }
    d := inc
    for i := 0; i < viper.GetInt("numcaps"); i++ {

        img, err := gen.Image(d)
        if err != nil {
            log.Fatalf("Can't generate screenshot: %v", err)
            os.Exit(1)
        }

        timestamp := fmt.Sprintf(time.Unix(d/1000, 0).UTC().Format("15:04:05"))
        log.Infof("generating screenshot %02d/%02d at %s", i+1, viper.GetInt("numcaps"), timestamp)
        var thumb image.Image
        if viper.GetInt("width") > 0 {
            thumb = imaging.Resize(img, viper.GetInt("width"), 0, imaging.Lanczos)
        } else if viper.GetInt("width") == 0 && viper.GetInt("height") > 0 {
            thumb = imaging.Resize(img, 0, viper.GetInt("height"), imaging.Lanczos)
        } else {
            thumb = img
        }

        //apply filters
        switch viper.GetString("filter") {
        case "greyscale":
            thumb = imaging.Grayscale(thumb)
            thumb = imaging.Sharpen(thumb, 1.0)
            thumb = imaging.AdjustContrast(thumb, 20)
            log.Debug("greyscale filter applied")
        case "invert":
            thumb = imaging.Invert(thumb)
            log.Debug("invert filter applied")
        }

        if !viper.GetBool("disable_timestamps") && !viper.GetBool("single_images") {
            log.Debug("adding timestamp to image")
            tsimage := drawTimestamp(timestamp)
            thumb = imaging.Overlay(thumb, tsimage, image.Pt(thumb.Bounds().Dx()-tsimage.Bounds().Dx()-10, thumb.Bounds().Dy()-tsimage.Bounds().Dy()-10), viper.GetFloat64("timestamp_opacity"))
        }

        //watermark middle image
        if i == (viper.GetInt("numcaps")-1)/2 && viper.GetString("watermark") != "" {
            ov, err := imaging.Open(viper.GetString("watermark"))
            if err == nil {
                if ov.Bounds().Dx() > thumb.Bounds().Dx() {
                    ov = imaging.Resize(ov, thumb.Bounds().Dx(), 0, imaging.Lanczos)
                }
                if ov.Bounds().Dy() > thumb.Bounds().Dy() {
                    ov = imaging.Resize(ov, 0, thumb.Bounds().Dy(), imaging.Lanczos)
                }
                posX := (thumb.Bounds().Dx() - ov.Bounds().Dx()) / 2
                posY := (thumb.Bounds().Dy() - ov.Bounds().Dy()) / 2
                thumb = imaging.Overlay(thumb, ov, image.Pt(posX, posY), 0.6)
            }
        }

        thumbnails = append(thumbnails, thumb)
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
    bgColor := strings.Split(viper.GetString("bg_content"), ",")
    var r, g, b int
    if len(bgColor) == 3 {
        r, _ = strconv.Atoi(strings.TrimSpace(bgColor[0]))
        g, _ = strconv.Atoi(strings.TrimSpace(bgColor[1]))
        b, _ = strconv.Atoi(strings.TrimSpace(bgColor[2]))
    } else {
        log.Info("useing fallback bg_content: 0,0,0")
        r, g, b = 0, 0, 0
    }
    dst := imaging.New(imgWidth*columns+paddingColumns, imgHeight*imgRows+paddingRows, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
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
        newIm := imaging.New(dst.Bounds().Dx(), dst.Bounds().Dy()+head.Bounds().Dy(), color.RGBA{uint8(r), uint8(g), uint8(b), 255})
        dst = imaging.Paste(newIm, dst, image.Pt(0, head.Bounds().Dy()))
        dst = imaging.Paste(dst, head, image.Pt(0, 0))
    }

    // save the combined image to file
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
    bgColor := strings.Split(viper.GetString("bg_header"), ",")
    var r, g, b int
    if len(bgColor) == 3 {
        r, _ = strconv.Atoi(strings.TrimSpace(bgColor[0]))
        g, _ = strconv.Atoi(strings.TrimSpace(bgColor[1]))
        b, _ = strconv.Atoi(strings.TrimSpace(bgColor[2]))
    } else {
        log.Debug("useing fallback bg_header: 0,0,0")
        r, g, b = 0, 0, 0
    }

    fgColor := strings.Split(viper.GetString("fg_header"), ",")
    var fr, fg, fb int
    if len(fgColor) == 3 {
        fr, _ = strconv.Atoi(strings.TrimSpace(fgColor[0]))
        fg, _ = strconv.Atoi(strings.TrimSpace(fgColor[1]))
        fb, _ = strconv.Atoi(strings.TrimSpace(fgColor[2]))
    } else {
        log.Debug("useing fallback bg_header: 255,255,255")
        fr, fg, fb = 255, 255, 255
    }

    fontcolor, bg := image.NewUniform(color.RGBA{uint8(fr), uint8(fg), uint8(fb), 255}), image.NewUniform(color.RGBA{uint8(r), uint8(g), uint8(b), 255})
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

    return []string{fname, fsize, duration, dimension}
}

var mpath string
var fontBytes []byte

func main() {
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
    viper.SetDefault("filename", "%s.jpg")
    viper.SetDefault("verbose", false)
    viper.SetDefault("bg_content", "0,0,0")
    viper.SetDefault("border", 0)
    viper.SetDefault("single_images", false)
    viper.SetDefault("header", true)
    viper.SetDefault("font_dirs", []string{})
    viper.SetDefault("bg_header", "0,0,0")
    viper.SetDefault("fg_header", "255,255,255")
    viper.SetDefault("header_image", "")
    viper.SetDefault("watermark", "")
    viper.SetDefault("filter", "none")

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

    flag.BoolP("disable-timestamps", "d", true, "disable-timestamps")
    viper.BindPFlag("disable_timestamps", flag.Lookup("disable-timestamps"))

    flag.BoolP("verbose", "v", true, "verbose output")
    viper.BindPFlag("verbose", flag.Lookup("verbose"))

    flag.BoolP("single-images", "s", true, "save single images instead of one combined contact sheet")
    viper.BindPFlag("single_images", flag.Lookup("single-images"))

    flag.String("bg-header", viper.GetString("bg_header"), "rgb background color for header")
    viper.BindPFlag("bg_header", flag.Lookup("bg-header"))

    flag.String("fg-header", viper.GetString("fg_header"), "rgb font color for header")
    viper.BindPFlag("fg_header", flag.Lookup("fg-header"))

    flag.String("bg-content", viper.GetString("bg_content"), "rgb background color for header")
    viper.BindPFlag("bg_content", flag.Lookup("bg-content"))

    flag.String("header-image", viper.GetString("header_image"), "image to put in the header")
    viper.BindPFlag("header_image", flag.Lookup("header-image"))

    viper.AutomaticEnv()

    viper.SetConfigType("json")
    viper.AddConfigPath("/etc/mt/")
    viper.AddConfigPath("$HOME/.mt")
    viper.AddConfigPath("./")

    err := viper.ReadInConfig()
    if err != nil {
        fmt.Errorf("error reading config file: %s useing default values", err)
    }

    flag.Parse()

    if len(flag.Args()) == 0 {
        flag.Usage()
        os.Exit(1)
    }

    if viper.GetBool("verbose") {
        log.SetLevel(log.DebugLevel)
    }

    fontBytes, err = getFont(viper.GetString("font_all"))
    if err != nil {
        log.Warn("unable to load font, disableing timestamps and header")
    }

    for _, movie := range flag.Args() {
        mpath = movie
        log.Infof("generating contact sheet for %s", movie)

        var thumbs []image.Image
        // thumbs = getImages(movie)
        // TODO: implement generation of image contac sheets from a folder
        thumbs = GenerateScreenshots(movie)
        if viper.GetBool("single_images") {
            for i, thumb := range thumbs {
                path, fname := filepath.Split(movie)
                newPath := filepath.Join(path, fmt.Sprintf("%s-%02d.jpg", fname, i+1))
                writeImage(thumb, newPath)
            }
        } else {
            makeContactSheet(thumbs, fmt.Sprintf(viper.GetString("filename"), movie))
        }

    }

}
