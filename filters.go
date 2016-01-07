package main

import (
	"bitbucket.org/raphaelmutschler/mt/Godeps/_workspace/src/github.com/disintegration/imaging"
	"bytes"
	"image"
	"image/color"
	"math"
)

func clamp(v float64) uint8 {
	return uint8(math.Min(math.Max(v, 0.0), 255.0) + 0.5)
}

func sigmoid(a, b, x float64) float64 {
	return 1 / (1 + math.Exp(b*(a-x)))
}

// ImageStrip look
func ImageStripFilter(img image.Image) image.Image {
	l, _ := Asset("strip_left.jpg")
	lr := bytes.NewReader(l)
	strip, _ := imaging.Decode(lr)
	strip = imaging.Resize(strip, 0, img.Bounds().Dy(), imaging.Lanczos)

	dst := imaging.New((2*strip.Bounds().Dx())+img.Bounds().Dx(), img.Bounds().Dy(), color.NRGBA{0, 0, 0, 0})
	dst = imaging.Paste(dst, img, image.Pt(strip.Bounds().Dx(), 0))
	dst = imaging.Paste(dst, strip, image.Pt(0, 0))

	r, _ := Asset("strip_right.jpg")
	rr := bytes.NewReader(r)
	stripr, _ := imaging.Decode(rr)
	stripr = imaging.Resize(stripr, 0, img.Bounds().Dy(), imaging.Lanczos)
	dst = imaging.Paste(dst, stripr, image.Pt(dst.Bounds().Dx()-stripr.Bounds().Dx(), 0))
	return dst
}

// sigmoid function to simulate image cross processing, best results with midpoint: 0.5 and factor 10
func CrossProcessingFilter(img image.Image, midpoint, factor float64) *image.NRGBA {

	red := make([]uint8, 256)
	green := make([]uint8, 256)
	blue := make([]uint8, 256)
	a := math.Min(math.Max(midpoint, 0.0), 1.0)
	b := math.Abs(factor)
	sig0 := sigmoid(a, b, 0)
	sig1 := sigmoid(a, b, 1)
	e := 1.0e-6

	for i := 0; i < 256; i++ {
		x := float64(i) / 255.0
		sigX := sigmoid(a, b, x)
		f := (sigX - sig0) / (sig1 - sig0)
		red[i] = clamp(f * 255.0)
	}

	for i := 0; i < 256; i++ {
		x := float64(i) / 255.0
		sigX := sigmoid(a, b, x)
		f := (sigX - sig0) / (sig1 - sig0)
		green[i] = clamp(f * 255.0)
	}

	for i := 0; i < 256; i++ {
		x := float64(i) / 255.0
		arg := math.Min(math.Max((sig1-sig0)*x+sig0, e), 1.0-e)
		f := a - math.Log(1.0/arg-1.0)/b
		blue[i] = clamp(f * 255.0)
	}

	fn := func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{red[c.R], green[c.G], blue[c.B], c.A}
	}

	return imaging.AdjustFunc(img, fn)
}

func xCrossProcessing(img image.Image, midpoint, factor float64) *image.NRGBA {

	red := make([]uint8, 256)
	green := make([]uint8, 256)
	blue := make([]uint8, 256)
	a := math.Min(math.Max(midpoint, 0.0), 1.0)
	b := math.Abs(factor)
	sig0 := sigmoid(a, b, 0)
	sig1 := sigmoid(a, b, 1)
	e := 1.0e-6

	for i := 0; i < 256; i++ {
		x := float64(i) / 255.0
		sigX := sigmoid(a, b, x)
		f := (sigX - sig0) / (sig1 - sig0)
		red[i] = clamp(f * 255.0)
	}

	for i := 0; i < 256; i++ {
		x := float64(i) / 255.0
		sigX := sigmoid(a, b, x)
		f := (sigX - sig0) / (sig1 - sig0)
		green[i] = clamp(f * 255.0)
	}

	for i := 0; i < 256; i++ {
		x := float64(i) / 255.0
		arg := math.Min(math.Max((sig1-sig0)*x+sig0, e), 1.0-e)
		f := a - math.Log(1.0/arg-1.0)/b
		blue[i] = clamp(f * 255.0)
	}

	fn := func(c color.NRGBA) color.NRGBA {
		//return color.NRGBA{blue[c.R], green[c.G], red[c.B], c.A}
		return color.NRGBA{blue[c.R], green[c.G], blue[c.B], c.A}
	}

	return imaging.AdjustFunc(img, fn)
}
