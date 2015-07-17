CGO_LDFLAGS=-L/usr/local/Cellar/ffmpeg/2.6.1/lib/
CGO_CFLAGS=-I/usr/local/Cellar/ffmpeg/2.6.1/include/

LD_LIBRARY_PATH=/usr/local/Cellar/ffmpeg/2.6.1/lib/

all:
	CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" LD_LIBRARY_PATH="$(LD_LIBRARY_PATH)" CGO_CXXFLAGS=$(CGO_CFLAGS) CGO_CPPFLAGS=$(CGO_CFLAGS) CFLAGS=$(CGO_CFLAGS) go build -v -a
	
