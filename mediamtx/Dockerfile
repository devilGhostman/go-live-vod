FROM mwader/static-ffmpeg AS ffmpeg

FROM golang:alpine3.21

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# addding ffmpeg
COPY --from=ffmpeg /ffmpeg /usr/local/bin/
COPY --from=ffmpeg /ffprobe /usr/local/bin/

RUN go build main.go

EXPOSE 9998 9999 1935 8888 8889 8890 9997

CMD [ "./main" ]
