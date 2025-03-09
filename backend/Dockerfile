FROM mwader/static-ffmpeg AS ffmpeg

FROM golang:alpine3.21

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# addding ffmpeg
COPY --from=ffmpeg /ffmpeg /usr/local/bin/
COPY --from=ffmpeg /ffprobe /usr/local/bin/

RUN go build cmd/main.go

EXPOSE 8000

CMD [ "./main" ]