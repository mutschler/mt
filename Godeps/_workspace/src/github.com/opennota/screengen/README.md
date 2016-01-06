screengen [![License](http://img.shields.io/:license-gpl3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0.html) [![GoDoc](https://godoc.org/github.com/opennota/screengen?status.svg)](http://godoc.org/github.com/opennota/screengen)
=========

A library for generating screenshots from video files, and a command-line tool for generating thumbnail grids.

## Install the package

    go get github.com/opennota/screengen

## Install the command-line tool

    go get github.com/opennota/screengen/cmd/screengen

Note: the package doesn't link against `libav`. So far, only `ffmpeg` is supported. Pull requests are welcome.
