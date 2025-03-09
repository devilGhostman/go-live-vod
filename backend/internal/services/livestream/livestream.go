package livestream

import (
	"context"
	"errors"
	"github/devilGhostman/backend/internal/databse"
	"github/devilGhostman/backend/internal/externalcmd"
	"github/devilGhostman/backend/internal/models"
	"github/devilGhostman/backend/internal/services/transcoder"
	"github/devilGhostman/backend/internal/types"
	"net/http"
	"os"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func Auth(c *gin.Context) {
	zap.S().Infof("livestream auth")
	c.JSON(http.StatusOK, gin.H{
		"message": "successfully authenticated",
	})
}

func StartHlsTranscode(c *gin.Context) {
	var rtmpBody types.RtmpConfigHTTP
	if err := c.ShouldBindJSON(&rtmpBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	zap.S().Infof("starting rtpm hls  %+v", rtmpBody)

	tsc := transcoder.NewTranscoder(
		rtmpBody.RtmpUrl,
		rtmpBody.Namespace,
		rtmpBody.ID,
		rtmpBody.Config.Resolutions,
		rtmpBody.Config.Codecs.Video,
		rtmpBody.Config.Codecs.Audio)

	tsc.SetConfig(rtmpBody.Config.Resolutions, true, rtmpBody.Config.Codecs.Video, rtmpBody.Config.Codecs.Audio)

	_, err := tsc.Run()
	if err != nil {
		zap.S().Errorf("failed to start trasncoder, Error: %s", err)
	}

	newLive, err := addLiveToDatabase(rtmpBody)
	if err != nil {
		zap.S().Errorf("failed to insert vod: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert vod"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "started livestream transcoding", "body": newLive})
}

func StopHlsTranscode(c *gin.Context) {
	id := c.Query("id")
	namespace := c.Query("namespace")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace is required"})
		return
	}

	zap.S().Infof("termanating running hls streaming ID:%s", id)

	activeCmd := externalcmd.GloblaActiveCmds[id]
	zap.S().Infof("total number of runnings hls streaming Count:%s \n current activeCmd: %s", len(externalcmd.GloblaActiveCmds), activeCmd)
	if activeCmd == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stream id not found"})
		return
	}
	defer activeCmd.Close()

	zap.S().Infof("closing httpproxy gracefully...\ncmdstring: %s \nprocessid: %s", activeCmd.GetCmdString(), activeCmd.GetProcess().Pid)
	syscall.Kill(-activeCmd.GetProcess().Pid, syscall.SIGINT)
	err := activeCmd.GetProcess().Kill()

	if err != nil {
		zap.S().Errorf("failed to terminate hls stream ID:%s", activeCmd.StreamID)
	}
	zap.S().Infof("closed processid: %s", activeCmd.GetProcess().Pid)
	delete(externalcmd.GloblaActiveCmds, id)

	zap.S().Infof("removing output dir: %s", "output"+"/"+namespace+"/"+id)
	os.RemoveAll("output" + "/" + namespace + "/" + id)

	c.JSON(http.StatusOK, gin.H{"message": "stopped livestream transcoding"})
}

func addLiveToDatabase(liveBody types.RtmpConfigHTTP) (models.Transcode, error) {
	liveCollection := databse.GetCollection("live")

	var newLive models.Transcode
	newLive.Title = liveBody.StreamName
	newLive.Status = "live"
	newLive.Config.Resolutions = liveBody.Config.Resolutions
	newLive.Config.Codecs.Video = liveBody.Config.Codecs.Video
	newLive.Config.Codecs.Audio = liveBody.Config.Codecs.Audio

	baseUrl := os.Getenv("BASE_URL")

	var hlsUrls models.VideoUrls
	hlsUrls.Master = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/playlist.m3u8"
	hlsUrls.Poster = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/thumbnail.jpg"

	for _, resolution := range liveBody.Config.Resolutions {
		switch resolution {
		case "1080p":
			hlsUrls.U1080p = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/1080p/1080p.m3u8"
		case "720p":
			hlsUrls.U720p = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/720p/720p.m3u8"
		case "480p":
			hlsUrls.U480p = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/480p/480p.m3u8"
		case "360p":
			hlsUrls.U360p = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/360p/360p.m3u8"
		case "240p":
			hlsUrls.U240p = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/240p/240p.m3u8"
		case "144p":
			hlsUrls.U144p = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/144p/144p.m3u8"
		}
	}

	hlsUrls.Audio = baseUrl + "/hls/" + liveBody.Namespace + "/" + liveBody.ID + "/audio/audio.m3u8"

	newLive.VideoUrls = hlsUrls

	dbRes, err := liveCollection.InsertOne(context.Background(), newLive)
	if err != nil {
		zap.S().Errorf("failed to insert vod: %s", err)
		return newLive, errors.New("failed to insert vod")
	}

	id, ok := dbRes.InsertedID.(primitive.ObjectID)
	if !ok {
		zap.S().Errorf("failed to get inserted id: %s", err)
		return newLive, errors.New("failed to get inserted id")

	}

	newLive.ID = id

	return newLive, nil
}

func GetAllLiveStreams(c *gin.Context) {
	liveCollection := databse.GetCollection("live")

	filter := bson.D{}
	findOptions := options.Find()

	ctx := c.Request.Context()

	cursor, err := liveCollection.Find(ctx, filter, findOptions)
	if err != nil {
		zap.S().Errorf("failed to get all live streams: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get all live streams"})
		return
	}
	defer cursor.Close(ctx)

	var allLive []bson.M

	if err := cursor.All(ctx, &allLive); err != nil {
		zap.S().Errorf("failed to decode live streams: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode live streams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    allLive,
	})
}
