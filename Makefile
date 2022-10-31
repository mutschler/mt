#target is to generate several bin files automatically
ARCHES := linux_amd64 i686-w64-mingw32 x86_64-w64-mingw32 arm-linux-gnueabihf

FFMPEG_PKG = ffmpeg-5.1.2
#FFMPEG_PKG = ffmpeg-4.4
FFMPEG_EXT = tar.bz2
FFMPEG_SRC = http://ffmpeg.org/releases/$(FFMPEG_PKG).$(FFMPEG_EXT)

GOARCH = $(shell go env GOARCH)
GOOS = $(shell go env GOOS)
PROJECTROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
PREFIX = $(PROJECTROOT)dep
FFMPEGTARGET = $(PREFIX)/ffmpeg_$(GOOS)_$(GOARCH)

ifeq ($(UNAME),Darwin)
	GOFLAGS = -ldflags '-L "$(PREFIX)/ffmpeg_$(GOOS)_$(GOARCH)/lib/" -extldflags "-static -Wl,--allow-multiple-definition"'
else
	GOFLAGS = -ldflags='-L "$(PREFIX)/ffmpeg_$(GOOS)_$(GOARCH)/lib/"'
endif

all: ffmpeg build
	echo $(FFMPEGTARGET)

build:
	go build $(GOFLAGS)
#	PKG_CONFIG_LIBDIR=$(FFMPEGTARGET)/lib/pkgconfig/ LD_LIBRARY_PATH=$(FFMPEGTARGET)/lib/ go build $(GOFLAGS)

buildffmpeg:
	mkdir -p $(FFMPEGTARGET)
	wget -P $(PREFIX) $(FFMPEG_SRC)
	tar -xf $(PREFIX)/$(FFMPEG_PKG).$(FFMPEG_EXT) -C $(PREFIX)/
	cd $(PREFIX)/$(FFMPEG_PKG) && ./configure --disable-yasm --disable-programs --disable-doc --prefix=$(FFMPEGTARGET)
	$(MAKE) -C $(PREFIX)/$(FFMPEG_PKG) --silent -j`nproc`
#;$(MAKE) -C $(PREFIX)/$(FFMPEG_PKG) --silent -j`nproc`
	$(MAKE) -C $(PREFIX)/$(FFMPEG_PKG)  install --silent
	zip -r $(PREFIX)/lib_ffmpeg_$(GOOS)_$(GOARCH) $(PREFIX)/ffmpeg_$(GOOS)_$(GOARCH)/lib/*


$(FFMPEGTARGET)/lib/libavcodec.a:
	$(MAKE) buildffmpeg

$(FFMPEGTARGET)/lib/libavformat.a:


$(FFMPEGTARGET)/lib/libavutil.a:


$(FFMPEGTARGET)/lib/libswresample.a:


$(FFMPEGTARGET)/lib/libswscale.a:


ffmpeg: $(FFMPEGTARGET)/lib/libavcodec.a $(FFMPEGTARGET)/lib/libavformat.a $(FFMPEGTARGET)/lib/libavutil.a $(FFMPEGTARGET)/lib/libswresample.a $(FFMPEGTARGET)/lib/libswscale.a


clean:
	rm -f $(PREFIX)/$(FFMPEG_PKG).$(FFMPEG_EXT)
	rm -rf $(PREFIX)/$(FFMPEG_PKG)
	rm -f mt

wipe: clean
	rm -rf dep

