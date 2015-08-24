package main

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

type config struct {
    Bg_Content         string `json:"bg_content"`
    Bg_Header          string `json:"bg_header"`
    Columns            int    `json:"columns"`
    Disable_Timestamps bool   `json:"disable_timestamps"`
    Fg_Header          string `json:"fg_header"`
    Filter             string `json:"filter"`
    Font_All           string `json:"font_all"`
    Font_Size          int    `json:"font_size"`
    Header             bool   `json:"header"`
    Header_Image       string `json:"header_image"`
    Header_Meta        bool   `json:"header_meta"`
	Numcaps            int    `json:"numcaps"`
    Filename           string `json:"filename"`
	Padding            int    `json:"padding"`
    Single_Images      bool   `json:"single_images"`
    Skip_Blank         bool   `json:"skip_blank"`
    Verbose            bool   `json:"verbose"`
	Width              int    `json:"width"`
	// Watermark          string `json:"watermark"`
}

var C config

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
