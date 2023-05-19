package main

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
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
	Font_All string `json:"font_all"`
	// FontSize is the font size to be used in the contact sheet.
	Font_Size int `json:"font_size"`
	// DisableTimestamps provides the ability to toggle timestamps in the
	// contact sheet.
	Disable_Timestamps bool `json:"disable_timestamps"`
	// Verbose increases the logging.
	Verbose bool `json:"verbose"`
	// SingleImages will create a single image for each screenshot.
	Single_Images bool `json:"single_images"`
	// BgHeader set the background color of the contact sheet header (RGB).
	Bg_Header string `json:"bg_header"`
	// FgHeader sets the foreground color of the contact sheet header (RGB).
	Fg_Header string `json:"fg_header"`
	// BgContent sets the background color of the contact sheet context (RGB).
	Bg_Content string `json:"bg_content"`
	// HeaderImage sets the contact sheet header to be an image.
	Header_Image string `json:"header_image"`
	// SkipBlank sets the ability to skip up to three blank images. Can impact
	// performance.
	Skip_Blank bool `json:"skip_blank"`
	// Header sets whether to create a header in the contact sheet.
	Header bool `json:"header"`
	// HeaderMeta sets whether to include codec, bitrate, and FPS to header.
	Header_Meta bool `json:"header_meta"` // Required header to be true?
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
	Skip_Existing bool `json:"skip_existing"`
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
	Watermark_All string `json:"watermark_all"`
	// Comment sets a line of text that will be displayed in the bottom-left corner
	// of the contact sheet header.
	Comment string `json:"comment"`
	// SkipBlurry sets the ability to skip up to three blurry images. Can impact
	// performance.
	Skip_Blurry bool `json:"skip_blurry"`
	// BlurThreshold sets the threshold for blur detection in thumbnails.
	Blur_Threshold int `json:"blur_threshold"`
	// BlankThreshold sets the threshold for blank detection in thumbnails.
	Blank_Threshold int `json:"blank_threshold"`
	// Webvtt generates a webtt file when enabled.
	Webvtt bool `json:"webvtt"`
	// Vtt ??
	Vtt bool `json:"vtt"`
	// Upload posts the generated contact sheet to a URL.
	Upload bool `json:"upload"`
	// UploadURL sets the upload URL.
	Upload_URL string `json:"upload_url"`
}

var C config
var tmpDir = ""

func saveConfig(configurationPath string) error {
	err := mapstructure.WeakDecode(viper.AllSettings(), &C)

	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(&C, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(configurationPath)
	if err != nil {
		return err
	}

	defer f.Close()

	f.WriteString(string(b))
	log.Infof("config file saved to: %s", configurationPath)

	return nil
}
