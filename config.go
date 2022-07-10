package main

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

type config struct {
	Numcaps            int    `json:"numcaps"`
	Columns            int    `json:"columns"`
	Padding            int    `json:"padding"`
	Width              int    `json:"width"`
	Font_All           string `json:"font_all"`
	Font_Size          int    `json:"font_size"`
	Disable_Timestamps bool   `json:"disable_timestamps"`
	Verbose            bool   `json:"verbose"`
	Single_Images      bool   `json:"single_images"`
	Bg_Header          string `json:"bg_header"`
	Fg_Header          string `json:"fg_header"`
	Bg_Content         string `json:"bg_content"`
	Header_Image       string `json:"header_image"`
	Skip_Blank         bool   `json:"skip_blank"`
	Header             bool   `json:"header"`
	Header_Meta        bool   `json:"header_meta"`
	Filter             string `json:"filter"`
	Filename           string `json:"filename"`
	From               string `json:"from"`
	To                 string `json:"to"`
	Skip_Existing      bool   `json:"skip_existing"`
	Overwrite          bool   `json:"overwrite"`
	SFW                bool   `json:"sfw"`
	Watermark          string `json:"watermark"`
	Fast               bool   `json:"fast"`
	Watermark_All      string `json:"watermark_all"`
	Comment            string `json:"comment"`
	Skip_Blurry        bool   `json:"skip_blurry"`
	Blur_Threshold     int    `json:"blur_threshold"`
	Blank_Threshold    int    `json:"blank_threshold"`
	Webvtt             bool   `json:"webvtt"`
	Vtt                bool   `json:"vtt"`
	Upload             bool   `json:"upload"`
	Upload_URL         string `json:"upload_url"`
}

var C config
var tmpDir = ""

func saveConfig(cfgpath string) error {
	err := mapstructure.WeakDecode(viper.AllSettings(), &C)

	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(&C, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(cfgpath)
	if err != nil {
		return err
	}

	defer f.Close()

	f.WriteString(string(b))
	log.Infof("config file saved to: %s", cfgpath)

	return nil
}
