#target is to generate several bin files automatically
ARCHES := linux_amd64 i686-w64-mingw32 x86_64-w64-mingw32 arm-linux-gnueabihf

FFMPEG_VERSION = 7.1.1
FFMPEG_PKG = ffmpeg-$(FFMPEG_VERSION)
FFMPEG_EXT = tar.bz2
FFMPEG_SRC = http://ffmpeg.org/releases/$(FFMPEG_PKG).$(FFMPEG_EXT)

GOARCH = $(shell go env GOARCH)
GOOS = $(shell go env GOOS)
PROJECTROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
PREFIX = $(PROJECTROOT)dep
FFMPEGTARGET = $(PREFIX)/ffmpeg_$(FFMPEG_VERSION)_$(GOOS)_$(GOARCH)
VERSIONFLAGS = -X main.GitVersion=`git describe --tags --always --dirty` -X main.BuildTimestamp=`date -u '+%Y-%m-%d_%I:%M:%S_UTC'` -X main.FfmpegVersion=$(FFMPEG_PKG)

ifeq ($(UNAME),Darwin)
	GOFLAGS = -ldflags "-s -w $(VERSIONFLAGS) -L '$(PREFIX)/ffmpeg_$(FFMPEG_VERSION)_$(GOOS)_$(GOARCH)/lib/' -extldflags '-static -Wl,--allow-multiple-definition'"
else
	GOFLAGS = -ldflags "-s -w $(VERSIONFLAGS) -L '$(PREFIX)/ffmpeg_$(FFMPEG_VERSION)_$(GOOS)_$(GOARCH)/lib/'"
endif

all: ffmpeg build

build:
	PKG_CONFIG_LIBDIR=$(FFMPEGTARGET)/lib/pkgconfig/ LD_LIBRARY_PATH=$(FFMPEGTARGET)/lib/ go build $(GOFLAGS)

buildffmpeg:
ifeq ("$(wildcard $(PREFIX)/$(FFMPEG_PKG).$(FFMPEG_EXT))","")
	mkdir -p $(FFMPEGTARGET)
endif
ifeq ("$(wildcard $(PREFIX)/$(FFMPEG_PKG).$(FFMPEG_EXT))","")
	wget -P $(PREFIX) $(FFMPEG_SRC)
endif
ifeq ("$(wildcard $(PREFIX)/$(FFMPEG_PKG))","")
	tar -xf $(PREFIX)/$(FFMPEG_PKG).$(FFMPEG_EXT) -C $(PREFIX)/
endif
	cd $(PREFIX)/$(FFMPEG_PKG) && ./configure --disable-yasm --disable-programs --disable-doc --prefix=$(FFMPEGTARGET)
	$(MAKE) -C $(PREFIX)/$(FFMPEG_PKG) --silent -j`nproc`;	$(MAKE) -C $(PREFIX)/$(FFMPEG_PKG) --silent -j`nproc`
	$(MAKE) -C $(PREFIX)/$(FFMPEG_PKG)  install --silent

$(FFMPEGTARGET)/lib/libavcodec.a:
	$(MAKE) buildffmpeg

$(FFMPEGTARGET)/lib/libavformat.a:
	$(MAKE) buildffmpeg

$(FFMPEGTARGET)/lib/libavutil.a:
	$(MAKE) buildffmpeg

$(FFMPEGTARGET)/lib/libswresample.a:
	$(MAKE) buildffmpeg

$(FFMPEGTARGET)/lib/libswscale.a:
	$(MAKE) buildffmpeg

ffmpeg: $(FFMPEGTARGET)/lib/libavcodec.a $(FFMPEGTARGET)/lib/libavformat.a $(FFMPEGTARGET)/lib/libavutil.a $(FFMPEGTARGET)/lib/libswresample.a $(FFMPEGTARGET)/lib/libswscale.a


clean:
	rm -f $(PREFIX)/$(FFMPEG_PKG).$(FFMPEG_EXT)
	rm -rf $(PREFIX)/$(FFMPEG_PKG)
	rm -f mt

wipe: clean
	rm -rf dep

