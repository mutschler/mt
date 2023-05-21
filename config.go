package main

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

const (
	configName = "mt"
	configType = "json"

	blurThreshold  = 62
	blankThreshold = 85
)

type config struct {
	// Numcaps is the number of thumbnails to use in the contact sheet.
	Numcaps int `json:"numcaps"`
	// Columns are how many columns should be used in the contact sheet.
	Columns int `json:"columns"`
	// Padding is how much padding (in pixels) to add around thumbnails in the
	// contact sheet.
	Padding int `json:"padding"`
	// Width is the width of a single thumbnail.
	Width int `json:"width"`
	// TODO is height missing?
	// FontAll is the font to use in the header and timestamps.
	FontAll string `json:"font_all"`
	// FontSize is the font size to be used in the contact sheet.
	FontSize int `json:"font_size"`
	// DisableTimestamps provides the ability to toggle timestamps in the
	// contact sheet.
	DisableTimestamps bool `json:"disable_timestamps"`
	// Verbose increases the logging.
	Verbose bool `json:"verbose"`
	// SingleImages will create a single image for each screenshot.
	SingleImages bool `json:"single_images"`
	// BgHeader set the background color of the contact sheet header (RGB).
	BgHeader string `json:"bg_header"`
	// FgHeader sets the foreground color of the contact sheet header (RGB).
	FgHeader string `json:"fg_header"`
	// BgContent sets the background color of the contact sheet context (RGB).
	BgContent string `json:"bg_content"`
	// HeaderImage sets the contact sheet header to be an image.
	HeaderImage string `json:"header_image"`
	// SkipBlank sets the ability to skip up to three blank images. Can impact
	// performance.
	SkipBlank bool `json:"skip_blank"`
	// Header sets whether to create a header in the contact sheet.
	Header bool `json:"header"`
	// HeaderMeta sets whether to include codec, bitrate, and FPS to header.
	HeaderMeta bool `json:"header_meta"` // Required header to be true?
	// Filter sets an optional filter on thumbnails. Options are:
	//   - "greyscale" greyscale color palette?
	//   - "invert"   invert image?
	//   - "fancy"     ?
	//   - "cross"     ?
	Filter string `json:"filter"`
	// Filename is the name of the contact sheet.
	Filename string `json:"filename"`
	// To is the starting timestamp.
	From string `json:"from"`
	// To is the ending timestamp.
	To string `json:"to"`
	// SkipExisting skips movie if there is and existing contact sheet.
	SkipExisting bool `json:"skip_existing"`
	// Overwrite will enable the ability to overwrite existing contact sheets.
	Overwrite bool `json:"overwrite"`
	// SFW enables nude detection (EXPERIMENTAL).
	SFW bool `json:"sfw"`
	// Watermark sets a provided image as the watermark in the content sections
	// of the contact sheet.
	Watermark string `json:"watermark"`
	// Fast enables faster creation of thumbnails. May result in duplicate screens.
	Fast bool `json:"fast"`
	// WatermarkAll sets a provided images as the watermark in each thumbnail.
	WatermarkAll string `json:"watermark_all"`
	// Comment sets a line of text that will be displayed in the bottom-left corner
	// of the contact sheet header.
	Comment string `json:"comment"`
	// SkipBlurry sets the ability to skip up to three blurry images. Can impact
	// performance.
	SkipBlurry bool `json:"skip_blurry"`
	// BlurThreshold sets the threshold for blur detection in thumbnails.
	BlurThreshold int `json:"blur_threshold"`
	// BlankThreshold sets the threshold for blank detection in thumbnails.
	BlankThreshold int `json:"blank_threshold"`
	// WebVTT generates a webtt file when enabled.
	WebVTT bool `json:"webvtt"`
	// VTT ??
	VTT bool `json:"vtt"`
	// Upload posts the generated contact sheet to a URL.
	Upload bool `json:"upload"`
	// UploadURL sets the upload URL.
	UploadUrl string `json:"upload_url"`
}

// configInit sets default variables and reads configuration file.
func configInit() {
	viper.AutomaticEnv()
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.SetEnvPrefix("mt")

	viper.AddConfigPath("./")
	viper.AddConfigPath("/etc/mt/")
	viper.AddConfigPath("$HOME/.mt")

	// Set mt defaults
	viper.SetDefault("numcaps", 4)
	viper.SetDefault("columns", 2)
	viper.SetDefault("padding", 10)
	viper.SetDefault("width", 400)
	viper.SetDefault("height", 0)
	viper.SetDefault("font_all", "DroidSans.ttf") // Should this be Ubuntu.ttf to match readme?
	viper.SetDefault("font_size", 12)
	viper.SetDefault("disable_timestamps", false)
	viper.SetDefault("timestamp_opacity", 1.0)
	viper.SetDefault("filename", "{{.Path}}{{.Name}}.jpg")
	viper.SetDefault("verbose", false)
	viper.SetDefault("bg_content", "0,0,0")
	viper.SetDefault("border", 0)
	viper.SetDefault("from", "00:00:00")
	viper.SetDefault("end", "00:00:00")
	viper.SetDefault("single_images", false)
	viper.SetDefault("header", true)
	viper.SetDefault("font_dirs", []string{})
	viper.SetDefault("bg_header", "0,0,0")
	viper.SetDefault("fg_header", "255,255,255")
	viper.SetDefault("header_image", "")
	viper.SetDefault("header_meta", false)
	viper.SetDefault("watermark", "")
	viper.SetDefault("comment", "contact sheet created with mt (https://github.com/mutschler/mt)")
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
	viper.SetDefault("vtt", false)
	viper.SetDefault("blur_threshold", blurThreshold)
	viper.SetDefault("blank_threshold", blankThreshold)
	viper.SetDefault("upload", false)
	viper.SetDefault("upload_url", "http://example.com/upload")
	viper.SetDefault("skip_credits", false)
	viper.SetDefault("interval", 0)

	err := viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		log.Info("configuration file not found, using defaults")
	}
	log.Info("loaded config file")

	// Bind values in config file to commandline flags
	flag.IntP("numcaps", "n", viper.GetInt("numcaps"), "number of captures to make")
	viper.BindPFlag("numcaps", flag.Lookup("numcaps"))

	flag.IntP("columns", "c", viper.GetInt("columns"), "number of columns")
	viper.BindPFlag("columns", flag.Lookup("columns"))

	flag.IntP("padding", "p", viper.GetInt("padding"), "padding between the images in px")
	viper.BindPFlag("padding", flag.Lookup("padding"))

	flag.IntP("width", "w", viper.GetInt("width"), "width of a single screenshot in px")
	viper.BindPFlag("width", flag.Lookup("width"))

	flag.StringP("font", "f", viper.GetString("font_all"), "font to use for timestamps and header information")
	viper.BindPFlag("font_all", flag.Lookup("font"))

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

	flag.Bool("webvtt", viper.GetBool("webvtt"), "create a .vtt file: disables header, header-meta, padding and timestamps")
	viper.BindPFlag("webvtt", flag.Lookup("webvtt"))

	flag.Bool("vtt", viper.GetBool("vtt"), "create a .vtt file for the generated image")
	viper.BindPFlag("vtt", flag.Lookup("vtt"))

	flag.Int("blur-threshold", viper.GetInt("blur_threshold"), "set a custom threshold to use for blurry image detection (defaults to 62)")
	viper.BindPFlag("blur_threshold", flag.Lookup("blur-threshold"))

	flag.Int("blank-threshold", viper.GetInt("blank_threshold"), "set a custom threshold to use for blank image detection (defaults to 85)")
	viper.BindPFlag("blank_threshold", flag.Lookup("blank-threshold"))

	flag.Bool("upload", viper.GetBool("upload"), "post file via http form submit")
	viper.BindPFlag("upload", flag.Lookup("upload"))

	flag.String("upload-url", viper.GetString("upload_url"), "url to use for --upload")
	viper.BindPFlag("upload_url", flag.Lookup("upload-url"))

	flag.IntP("interval", "i", viper.GetInt("interval"), "interval in seconds to take screencaps from, overwrites numcaps (defaults to 0)")
	viper.BindPFlag("interval", flag.Lookup("interval"))

	flag.Bool("skip-credits", viper.GetBool("skip_credits"), "tries to skip ending credits from screencap creation by cutting off 4 minutes or 10 percent of the clip (defaults to false)")
	viper.BindPFlag("skip_credits", flag.Lookup("skip-credits"))

	flag.Parse()
}

func saveConfig(configurationPath string) error {
	var currentConfig config
	err := mapstructure.WeakDecode(viper.AllSettings(), &currentConfig)

	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(&currentConfig, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(configurationPath)
	if err != nil {
		return ErrCannotSaveConfigFile
	}

	defer f.Close()

	f.WriteString(string(b))
	log.Infof("config file saved to: %s", configurationPath)

	return nil
}
