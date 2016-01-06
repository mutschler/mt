// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General
// Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package screengen can be used for generating screenshots from video files.
package screengen

// #cgo pkg-config: libavcodec libavformat libavutil libswscale
// #include <stdlib.h>
// #include <libavcodec/avcodec.h>
// #include <libavformat/avformat.h>
// #include <libswscale/swscale.h>
// #include <libavutil/log.h>
import "C"

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"unsafe"
)

// Generator is used to generate screenshots from a video file.
type Generator struct {
	Fast bool // Imprecise (but faster) seek; set by the user

	Filename           string  // Video file name
	Width              int     // Width of the video
	Height             int     // Height of the video
	Duration           int64   // Duration of the video in milliseconds
	VideoCodec         string  // Name of the video codec
	VideoCodecLongName string  // Readable/long name of the video codec
	FPS                float64 // Frames Per Second
	numberOfStreams    int
	AudioCodec         string // Name of the audio codec
	AudioCodecLongName string // Readable/long name of the audio codec
	vStreamIndex       int
	aStreamIndex       int
	Bitrate            int
	streams            []*C.struct_AVStream
	avfContext         *C.struct_AVFormatContext
	avcContext         *C.struct_AVCodecContext
}

// NewGenerator returns new generator of screenshots for the video file fn.
func NewGenerator(fn string) (_ *Generator, err error) {
	avfCtx := C.avformat_alloc_context()
	cfn := C.CString(fn)
	defer C.free(unsafe.Pointer(cfn))
	if C.avformat_open_input(&avfCtx, cfn, nil, nil) != 0 {
		return nil, errors.New("can't open input stream")
	}
	defer func() {
		if err != nil {
			C.avformat_close_input(&avfCtx)
		}
	}()
	if C.avformat_find_stream_info(avfCtx, nil) < 0 {
		return nil, errors.New("can't get stream info")
	}
	duration := int64(avfCtx.duration) / 1000
	bitrate := int(avfCtx.bit_rate) / 1000
	numberOfStreams := int(avfCtx.nb_streams)
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(avfCtx.streams)),
		Len:  numberOfStreams,
		Cap:  numberOfStreams,
	}
	streams := *(*[]*C.struct_AVStream)(unsafe.Pointer(&hdr))
	vStreamIndex := -1
	aStreamIndex := -1
	for i := 0; i < numberOfStreams; i++ {
		if streams[i].codec.codec_type == C.AVMEDIA_TYPE_VIDEO {
			vStreamIndex = i
		} else if streams[i].codec.codec_type == C.AVMEDIA_TYPE_AUDIO {
			aStreamIndex = i
		}
	}
	if vStreamIndex == -1 {
		return nil, errors.New("no video stream")
	}
	avcCtx := streams[vStreamIndex].codec
	vCodec := C.avcodec_find_decoder(avcCtx.codec_id)
	if vCodec == nil {
		return nil, errors.New("can't find decoder")
	}
	if C.avcodec_open2(avcCtx, vCodec, nil) != 0 {
		return nil, errors.New("can't initialize codec context")
	}
	width := int(avcCtx.width)
	height := int(avcCtx.height)
	fps := (float64(streams[vStreamIndex].r_frame_rate.num) /
		float64(streams[vStreamIndex].r_frame_rate.den))
	vCodecName := strings.ToUpper(C.GoString(vCodec.name))
	vCodecHuman := C.GoString(vCodec.long_name)

	aCodecName := ""
	aCodecHuman := ""
	if aStreamIndex != -1 {
		aacCtx := streams[aStreamIndex].codec
		aCodec := C.avcodec_find_decoder(aacCtx.codec_id)
		if aCodec != nil {
			aCodecName = strings.ToUpper(C.GoString(aCodec.name))
			aCodecHuman = C.GoString(aCodec.long_name)
		}
	}

	return &Generator{
		Filename:           fn,
		Width:              width,
		Height:             height,
		Duration:           duration,
		VideoCodec:         vCodecName,
		VideoCodecLongName: vCodecHuman,
		AudioCodec:         aCodecName,
		AudioCodecLongName: aCodecHuman,
		numberOfStreams:    numberOfStreams,
		vStreamIndex:       vStreamIndex,
		aStreamIndex:       aStreamIndex,
		FPS:                fps,
		Bitrate:            bitrate,
		streams:            streams,
		avfContext:         avfCtx,
		avcContext:         avcCtx,
	}, nil
}

// Image returns a screenshot at the ts milliseconds.
func (g *Generator) Image(ts int64) (image.Image, error) {
	return g.ImageWxH(ts, g.Width, g.Height)
}

// ImageWxH returns a screenshot at the ts milliseconds, scaled to the specified width and height.
func (g *Generator) ImageWxH(ts int64, width, height int) (image.Image, error) {
	frameNum := C.av_rescale(
		C.int64_t(ts),
		C.int64_t(g.streams[g.vStreamIndex].time_base.den),
		C.int64_t(g.streams[g.vStreamIndex].time_base.num),
	) / 1000
	if C.avformat_seek_file(
		g.avfContext,
		C.int(g.vStreamIndex),
		0,
		frameNum,
		frameNum,
		C.AVSEEK_FLAG_FRAME,
	) < 0 {
		if C.avformat_seek_file(
			g.avfContext,
			C.int(g.vStreamIndex),
			0,
			frameNum,
			frameNum,
			C.AVSEEK_FLAG_ANY,
		) < 0 {
			return nil, errors.New("can't seek to timestamp")
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	frame := C.av_frame_alloc()
	defer C.av_frame_free(&frame)
	C.avcodec_flush_buffers(g.avcContext)
	var pkt C.struct_AVPacket
	var frameFinished C.int
	for C.av_read_frame(g.avfContext, &pkt) == 0 {
		if int(pkt.stream_index) != g.vStreamIndex {
			C.av_free_packet(&pkt)
			continue
		}
		if C.avcodec_decode_video2(g.avcContext, frame, &frameFinished, &pkt) <= 0 {
			C.av_free_packet(&pkt)
			continue
		}
		C.av_free_packet(&pkt)
		if frameFinished == 0 || (!g.Fast && pkt.dts < frameNum) {
			continue
		}
		ctx := C.sws_getContext(
			C.int(g.Width),
			C.int(g.Height),
			g.avcContext.pix_fmt,
			C.int(width),
			C.int(height),
			C.PIX_FMT_RGBA,
			C.SWS_BICUBIC,
			nil,
			nil,
			nil,
		)
		if ctx == nil {
			return nil, errors.New("can't allocate scaling context")
		}
		srcSlice := (**C.uint8_t)(&frame.data[0])
		srcStride := (*C.int)(&frame.linesize[0])
		dst := (**C.uint8_t)(unsafe.Pointer(&img.Pix))
		dstStride := (*C.int)(unsafe.Pointer(&[1]int{img.Stride}))
		C.sws_scale(
			ctx,
			srcSlice,
			srcStride,
			0,
			g.avcContext.height,
			dst,
			dstStride,
		)
		break
	}
	return img, nil
}

// Close closes the internal ffmpeg context.
func (g *Generator) Close() error {
	C.avformat_close_input(&g.avfContext)
	return nil
}

func init() {
	C.av_log_set_level(C.AV_LOG_QUIET)
	C.av_register_all()
}
