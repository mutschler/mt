package main

import (
    "flag"
    "fmt"
    "github.com/disintegration/imaging"
    "github.com/cytec/screengen"
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
    // "github.com/spf13/cobra"
    "github.com/spf13/viper"
    // "code.google.com/p/freetype-go/freetype"
    // "code.google.com/p/freetype-go/freetype/truetype"
    //
    "code.google.com/p/jamslam-freetype-go/freetype"
    "code.google.com/p/jamslam-freetype-go/freetype/truetype"

    "github.com/dustin/go-humanize"
)

// trys to find a font in the local FS
func findFont(f string) string {
    if strings.Contains(f, "/") && strings.HasSuffix(f, ".ttf") {
        return f
    }
    fdirs := []string{"/Library/Fonts/", "/usr/share/fonts/", "./"}

    if !strings.HasSuffix(f, ".ttf") {
        f = fmt.Sprintf("%s.ttf", f)
    }
    for _, dir := range fdirs {
        fpath := filepath.Join(dir, f)
        if _, err := os.Stat(fpath); err == nil {
            return fpath
        }
    }
    return ""
}

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

    fontBytes, err := ioutil.ReadFile(findFont(viper.GetString("font_all")))

    if err != nil {
        fmt.Println(err)
        return timestamped
    }
    font, err := freetype.ParseFont(fontBytes)
    if err != nil {
        fmt.Println(err)
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
        fmt.Println(err)
        return timestamped
    }
    return rgba

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

    inc := gen.Duration / int64(viper.GetInt("numcaps"))
    if inc <= 60000 {
        fmt.Println("verry small timestamps in use... consider decreasing numcaps")
    }
    d := inc
    for i := 0; i < viper.GetInt("numcaps"); i++ {
        img, err := gen.Image(d)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Can't generate screenshot: %v\n", err)
            os.Exit(1)
        }

        // fn := filepath.Join("/", fmt.Sprintf(viper.GetString("filename"), fn, i))
        //writeImage(img, fn)
        timestamp := fmt.Sprintf(time.Unix(d/1000, 0).UTC().Format("15:04:05"))
        fmt.Printf("generating screenshot %02d/%02d at %s\n", i+1, viper.GetInt("numcaps"), timestamp)
        var thumb image.Image
        if viper.GetInt("width") > 0 {
            thumb = imaging.Resize(img, viper.GetInt("width"), 0, imaging.Lanczos)
        } else if viper.GetInt("width") == 0 && viper.GetInt("height") > 0 {
            thumb = imaging.Resize(img, 0, viper.GetInt("height"), imaging.Lanczos)
        }

        if !viper.GetBool("disable_timestamps") && !viper.GetBool("single_images") {
            tsimage := drawTimestamp(timestamp)
            thumb = imaging.Paste(thumb, tsimage, image.Pt(thumb.Bounds().Dx()-tsimage.Bounds().Dx()-10, thumb.Bounds().Dy()-tsimage.Bounds().Dy()-10))
            //fmt.Sprintf(time.Unix(d/1000, 0).UTC().Format("15:04:05"))
        }

        thumbnails = append(thumbnails, thumb)
        d += inc
    }

    return thumbnails
}

func makeContactSheet(thumbs []image.Image, fn string) {
    fmt.Println("Composing Contact Sheet")
    imgWidth := thumbs[0].Bounds().Dx()
    imgHeight := thumbs[0].Bounds().Dy()

    columns := viper.GetInt("columns")
    imgRows := int(math.Ceil(float64(len(thumbs)) / float64(columns)))

    if viper.GetBool("verbose") {
        fmt.Printf("Single Image: %dx%d\n", imgWidth, imgHeight)
        fmt.Printf("New Image: %dx%d\n", imgWidth*columns, imgHeight*imgRows)
    }

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
        fmt.Println("useing fallback bg_content: 0,0,0")
        r, g, b = 0, 0, 0
    }
    dst := imaging.New(imgWidth*columns+paddingColumns, imgHeight*imgRows+paddingRows, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
    x := 0
    curRow := 0
    // paste thumbnails into the new image side by side
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
        fmt.Println("appending header informations")
        head := appendHeader(dst)
        newIm := imaging.New(dst.Bounds().Dx(), dst.Bounds().Dy() + head.Bounds().Dy(), color.RGBA{uint8(r), uint8(g), uint8(b), 255} )
        dst = imaging.Paste(newIm, dst, image.Pt(0, head.Bounds().Dy()))
        dst = imaging.Paste(dst, head, image.Pt(0, 0))
    }

    // save the combined image to file
    err := imaging.Save(dst, fn)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Saved to %s\n", fn)
}

func appendHeader(im image.Image) image.Image {
    var timestamped image.Image
    fontBytes, err := ioutil.ReadFile(findFont(viper.GetString("font_all")))

    if err != nil {
        fmt.Println(err)
        return timestamped
    }
    font, err := freetype.ParseFont(fontBytes)
    if err != nil {
        fmt.Println(err)
        return timestamped
    }

    fg, bg := image.White, image.Black
    c := freetype.NewContext()
    c.SetDPI(72)
    c.SetFont(font)
    c.SetFontSize(float64(viper.GetInt("font_size")))

    // get width and height of the string and draw an image to hold it
    //x, y, _ := c.MeasureString(timestamp)
    rgba := image.NewRGBA(image.Rect(0, 0, im.Bounds().Dx(), (5+int(c.PointToFix32(float64(viper.GetInt("font_size")+2))>>8)*6) + 10))
    draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
    c.SetClip(rgba.Bounds())
    c.SetDst(rgba)
    c.SetSrc(fg)

    
    
    header := createHeader(mpath)
    for i, s := range header {
        //draw the text with 5px padding and lineheight +2 
        pt := freetype.Pt(5, (5+int(c.PointToFix32(float64(viper.GetInt("font_size")+2))>>8)*(i+1)))
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
    fsize := fmt.Sprintf("Size: %s", humanize.Bytes(uint64(stat.Size())))
    fname = fmt.Sprintf("Filename: %s", fname)
    // fmt.Println(fsize, fname)

    gen, err := screengen.NewGenerator(fn)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading video file: %v\n", err)
        os.Exit(1)
    }
    defer gen.Close()

    //skip 4 minutes of duration to remove intro and credits
    duration := fmt.Sprintf("Duration: %s", time.Unix(gen.Duration/1000, 0).UTC().Format("15:04:05"))
    // fmt.Println(duration)

    dimension := fmt.Sprintf("Dimensions: %dx%d", gen.Width, gen.Height)
    fps := fmt.Sprintf("FPS: %f", gen.FPS)

    // fmt.Println(gen.VideoCodec)
    codec := fmt.Sprintf("Codec: %s", gen.CodecName)

    // fmt.Println(dimension, fps)
    // fmt.Sprintf("%s \n %s \n %s", fname, fsize, duration)
    return []string{fname, fsize, duration, dimension, codec, fps}
    //os.Exit(1)
}

var mpath string

func main() {
    viper.SetConfigName("mt")
    viper.SetEnvPrefix("mt")
    viper.SetDefault("numcaps", 24)
    viper.SetDefault("columns", 4)
    viper.SetDefault("padding", 0)
    viper.SetDefault("width", 400)
    viper.SetDefault("height", 160)
    viper.SetDefault("font_all", "Arial.ttf")
    viper.SetDefault("font_size", 12)
    viper.SetDefault("disable_timestamps", false)
    viper.SetDefault("filename", "%s.jpg")
    viper.SetDefault("verbose", false)
    viper.SetDefault("bg_content", "0,0,0")
    viper.SetDefault("border", 0)
    viper.SetDefault("single_images", false)
    viper.SetDefault("header", true)
    viper.SetDefault("font_dirs", []string{})

    viper.AutomaticEnv()

    viper.SetConfigType("json")
    viper.AddConfigPath("/etc/mt/")
    viper.AddConfigPath("$HOME/.mt")
    viper.AddConfigPath("./")

    err := viper.ReadInConfig() // Find and read the config file
    if err != nil {             // Handle errors reading the config file
        //panic(fmt.Errorf("Fatal error config file: %s \n", err))
        fmt.Errorf("error reading config file: %s useing default values\n", err)
    }

    flag.Parse()

    if len(flag.Args()) == 0 {
        flag.Usage()
        os.Exit(1)
    }

    for _, movie := range flag.Args() {
        mpath = movie
        fmt.Printf("generating contact sheet for %s\n", movie)
        if strings.Contains(viper.GetString("font_all"), "/") {
            fmt.Printf("useing font: %s\n", viper.GetString("font_all"))
        } else {
            if findFont(viper.GetString("font_all")) == "" {
                fmt.Printf("unable to find font %s, timestamps disabled\n", viper.GetString("font_all"))
                viper.Set("disable_timestamps", true)
            } else {
                fmt.Printf("useing font found at: %s\n", findFont(viper.GetString("font_all")))
            }

        }

        thumbs := GenerateScreenshots(movie)
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
