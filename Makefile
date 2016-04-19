#target is to generate several bin files automatically
ARCHES := linux_amd64 i686-w64-mingw32 x86_64-w64-mingw32 arm-linux-gnueabihf

ffmpeg:
	wget -P /tmp http://ffmpeg.org/releases/ffmpeg-3.0.1.tar.bz2
	tar -xvf /tmp/ffmpeg-3.0.1.tar.bz2 -C /tmp/
	cd /tmp/ffmpeg-3.0.1 && ./configure --disable-yasm --disable-programs --disable-doc --prefix=/tmp/ffmpeg && make && make install

all: ffmpeg
	PKG_CONFIG_LIBDIR=/tmp/ffmpeg/lib/pkgconfig/ LD_LIBRARY_PATH=/tmp/ffmpeg/lib/ go build --ldflags '-extldflags "-static -Wl,--allow-multiple-definition"'
