# GO-VOD-LIVE-TRANSCODING


## Getting started
### Prerequisites

**FFMPEG**


### Cloning the repository

```shell
https://github.com/devilGhostman/go-live-vod.git
```


## Use Docker Compose 
### Create and start the container
- From the root of the repository run the command:
```shell
docker-compose up -d --build
```
### Stop and remove the container
```shell
docker compose down --rmi "all"    
```

## To Run Live Stream
```shell
ffmpeg -re -stream_loop -1 -i path_to_your_video -c:v libx264 -c:a aac -f flv rtmp://localhost/*environment/*someId
```
### Example
```shell
ffmpeg -re -stream_loop -1 -i /test.mp4 -c:v libx264 -c:a aac -f flv rtmp://localhost/dev/12345678
```

