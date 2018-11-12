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
// #include <libavutil/mathematics.h>
//
// const int AVERROR_EAGAIN = AVERROR(EAGAIN);
//
// // Work around the Cgo pointer passing rules introduced in Go 1.6.
// int sws_scale_wrapper(
// 			struct SwsContext *c,
// 			const uint8_t *const srcSlice[],
// 			const int srcStride[],
// 			int srcSliceY,
// 			int srcSliceH,
// 			uint8_t dst[],
// 			const int dstStride[]
// 			) {
// 	return sws_scale(c, srcSlice, srcStride, srcSliceY, srcSliceH, &dst, dstStride);
// }
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
	width              int     // Width of the video
	height             int     // Height of the video
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
	Orientation        Orientation
	streams            []*C.struct_AVStream
	avfContext         *C.struct_AVFormatContext
	avcContext         *C.struct_AVCodecContext
}

type Orientation int

const (
	AVIdentity Orientation = 0

	AVRotation90 = 1 << iota
	AVRotation180
	AVRotation270
	AVRotationCustom
	AVFlipHorizontal
	AVFlipVertical
)

// Width returns the width of the video
func (g *Generator) Width() int { return g.width }

// Height returns the height of the video
func (g *Generator) Height() int { return g.height }

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
	fps := (float64(streams[vStreamIndex].avg_frame_rate.num) /
		float64(streams[vStreamIndex].avg_frame_rate.den))
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

	displayMatrix := C.av_stream_get_side_data(streams[vStreamIndex], C.AV_PKT_DATA_DISPLAYMATRIX, nil)
	var orientation Orientation
	if displayMatrix != nil {
		hdr := reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(displayMatrix)),
			Len:  9,
			Cap:  9,
		}
		matrix := *(*[]C.int32_t)(unsafe.Pointer(&hdr))
		det := matrix[0]*matrix[4] - matrix[1]*matrix[3]
		matrixCopy := make([]C.int32_t, 9)
		copy(matrixCopy, matrix)
		matrix = matrixCopy
		if det < 0 {
			orientation = AVFlipHorizontal
			flip := []int{-1, 0, 1}
			for i := 0; i < 9; i++ {
				matrix[i] *= C.int32_t(flip[i%3])
			}
		}
		if matrix[1] == 1<<16 && matrix[3] == -(1<<16) {
			orientation |= AVRotation90
		} else if matrix[0] == -(1<<16) && matrix[4] == -(1<<16) {
			if det < 0 {
				orientation = AVFlipVertical
			} else {
				orientation |= AVRotation180
			}
		} else if matrix[1] == -(1<<16) && matrix[3] == 1<<16 {
			orientation |= AVRotation270
		} else if matrix[0] == 1<<16 && matrix[4] == 1<<16 {
			orientation |= AVIdentity
		} else {
			orientation |= AVRotationCustom
		}
	}
	if orientation == AVRotation90 || orientation == AVRotation270 {
		width, height = height, width
	}

	return &Generator{
		Filename:           fn,
		width:              width,
		height:             height,
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
		Orientation:        orientation,
		streams:            streams,
		avfContext:         avfCtx,
		avcContext:         avcCtx,
	}, nil
}

// Image returns a screenshot at the ts milliseconds.
func (g *Generator) Image(ts int64) (image.Image, error) {
	return g.ImageWxH(ts, g.width, g.height)
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
	gw, gh := g.width, g.height
	if o := g.Orientation; o == AVRotation90 || o == AVRotation270 {
		width, height = height, width
		gw, gh = gh, gw
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	frame := C.av_frame_alloc()
	defer C.av_frame_free(&frame)
	C.avcodec_flush_buffers(g.avcContext)
	pkt := C.av_packet_alloc()
	for C.av_read_frame(g.avfContext, pkt) == 0 {
		if int(pkt.stream_index) != g.vStreamIndex {
			C.av_packet_unref(pkt)
			continue
		}
		if C.avcodec_send_packet(g.avcContext, pkt) != 0 {
			C.av_packet_unref(pkt)
			return nil, errors.New("avcodec_send_packet failed")
		}
		dts := pkt.dts
		C.av_packet_unref(pkt)
		if ret := C.avcodec_receive_frame(g.avcContext, frame); ret != 0 {
			if ret != C.AVERROR_EAGAIN {
				return nil, errors.New("avcodec_receive_frame failed")
			}
			continue
		}
		if !g.Fast && dts < frameNum {
			continue
		}
		ctx := C.sws_getContext(
			C.int(gw),
			C.int(gh),
			g.avcContext.pix_fmt,
			C.int(width),
			C.int(height),
			C.AV_PIX_FMT_RGBA,
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
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&img.Pix))
		dst := (*C.uint8_t)(unsafe.Pointer(hdr.Data))
		dstStride := (*C.int)(unsafe.Pointer(&[1]int{img.Stride}))
		C.sws_scale_wrapper(
			ctx,
			srcSlice,
			srcStride,
			0,
			g.avcContext.height,
			dst,
			dstStride,
		)
		C.sws_freeContext(ctx)
		break
	}

	switch g.Orientation {
	case AVRotation90:
		img = rotate90(img)
	case AVRotation180:
		img = rotate180(img)
	case AVRotation270:
		img = rotate270(img)
	}

	return img, nil
}

// Close closes the internal ffmpeg context.
func (g *Generator) Close() error {
	C.avcodec_close(g.avcContext)
	C.avformat_close_input(&g.avfContext)
	return nil
}

func init() {
	C.av_log_set_level(C.AV_LOG_QUIET)
	C.av_register_all()
}
