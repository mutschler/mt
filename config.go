package main

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
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

var tmpDir = ""

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
