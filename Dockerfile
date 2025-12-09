FROM golang:alpine

WORKDIR /app

RUN apk add ffmpeg-dev musl-dev gcc make

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /bin/mt

RUN apk del musl-dev gcc make

CMD ["--help"]
ENTRYPOINT ["/bin/mt"]