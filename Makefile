#target is to generate several bin files automatically
ARCHES := linux_amd64 i686-w64-mingw32 x86_64-w64-mingw32 arm-linux-gnueabihf

FFMPEG_PKG = ffmpeg-4.4
FFMPEG_EXT = tar.bz2
FFMPEG_SRC = http://ffmpeg.org/releases/$(FFMPEG_PKG).$(FFMPEG_EXT)

ifeq ($(UNAME),Darwin)
	GOFLAGS = --ldflags '-extldflags "-static -Wl,--allow-multiple-definition"'
else
	GOFLAGS =
endif

all: ffmpeg
	PKG_CONFIG_LIBDIR=/tmp/ffmpeg/lib/pkgconfig/ LD_LIBRARY_PATH=/tmp/ffmpeg/lib/ go build $(GOFLAGS)

ffmpeg:
	wget -P /tmp $(FFMPEG_SRC)
	tar -xf /tmp/$(FFMPEG_PKG).$(FFMPEG_EXT) -C /tmp/
	cd /tmp/$(FFMPEG_PKG) && ./configure --disable-yasm --disable-programs --disable-doc --prefix=/tmp/ffmpeg && make --silent -j`nproc` && make install --silent

clean:
	rm /tmp/$(FFMPEG_PKG).$(FFMPEG_EXT)
	rm -rf /tmp/$(FFMPEG_PKG)
