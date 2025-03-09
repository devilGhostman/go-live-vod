package videoOnDemand

import (
	"context"
	"encoding/json"
	"errors"
	"github/devilGhostman/backend/internal/databse"
	"github/devilGhostman/backend/internal/models"
	rabbitmq "github/devilGhostman/backend/internal/rabbitMq"
	"github/devilGhostman/backend/internal/services/transcoder"
	"github/devilGhostman/backend/internal/types"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Message struct {
	Action  string              `json:"Action"`
	Body    types.VodConfigHTTP `json:"Body"`
	MongoId primitive.ObjectID  `json:"MongoId"`
}

func StartVodTranscode(c *gin.Context) {
	var vodBody types.VodConfigHTTP
	if err := c.ShouldBindJSON(&vodBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, newVod, err := addVodToDatabase(vodBody)
	if err != nil {
		zap.S().Errorf("failed to insert vod: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert vod"})
		return
	}

	err = sendMesssageToRabbitMq(id, vodBody)
	if err != nil {
		zap.S().Errorf("failed to send message to rabbitmq: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message to rabbitmq"})
		return
	}

	zap.S().Infof("starting rtpm hls  %+v", vodBody)

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"body":    newVod,
	})

}

func addVodToDatabase(vodBody types.VodConfigHTTP) (primitive.ObjectID, models.Transcode, error) {
	vodCollection := databse.GetCollection("vod")
	var newVod models.Transcode
	newVod.Title = vodBody.StreamName
	newVod.Status = "pending"
	newVod.Config.Resolutions = vodBody.Config.Resolutions
	newVod.Config.Codecs.Video = vodBody.Config.Codecs.Video
	newVod.Config.Codecs.Audio = vodBody.Config.Codecs.Audio

	dbRes, err := vodCollection.InsertOne(context.Background(), newVod)
	if err != nil {
		zap.S().Errorf("failed to insert vod: %s", err)
		return primitive.NilObjectID, newVod, errors.New("failed to insert vod")
	}
	id, ok := dbRes.InsertedID.(primitive.ObjectID)
	if !ok {
		zap.S().Errorf("failed to get inserted id: %s", err)
		return primitive.NilObjectID, newVod, errors.New("failed to get inserted id")

	}
	newVod.ID = id

	return id, newVod, nil
}

func sendMesssageToRabbitMq(id primitive.ObjectID, vodBody types.VodConfigHTTP) error {
	rabbitMq := rabbitmq.RabbitMQInstance
	message := struct {
		Action  string
		Body    types.VodConfigHTTP
		MongoId primitive.ObjectID
	}{
		Action:  "TO_START_VOD_TRANSCODING",
		Body:    vodBody,
		MongoId: id,
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		zap.S().Errorf("failed to marshal message: %s", err)
		return err
	}

	err = rabbitMq.PublishMessage(rabbitMq.Channel, jsonMessage)
	if err != nil {
		zap.S().Errorf("failed to publish message: %s", err)
		return err
	}
	return nil
}

func StartVodTranscoding(messsage []byte) error {

	var vodBody Message
	err := json.Unmarshal(messsage, &vodBody)
	if err != nil {
		zap.S().Errorf("failed to unmarshal message: %s", err)
		return err
	}

	zap.S().Infof("starting rtpm hls  %+v", vodBody)

	tsc := transcoder.NewVodTranscoder(
		vodBody.Body.SourceUrl,
		vodBody.Body.Namespace,
		vodBody.Body.ID,
		vodBody.Body.Config.Resolutions,
		vodBody.Body.Config.Codecs.Video,
		vodBody.Body.Config.Codecs.Audio)

	err = tsc.Run()
	if err != nil {
		zap.S().Errorf("failed to start trasncoder, Error: %s", err)
		return err
	}

	baseUrl := os.Getenv("BASE_URL")

	var hlsUrls models.VideoUrls
	hlsUrls.Master = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/playlist.m3u8"
	hlsUrls.Poster = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/thumbnail.jpg"

	for _, resolution := range vodBody.Body.Config.Resolutions {
		switch resolution {
		case "1080p":
			hlsUrls.U1080p = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/1080p/1080p.m3u8"
		case "720p":
			hlsUrls.U720p = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/720p/720p.m3u8"
		case "480p":
			hlsUrls.U480p = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/480p/480p.m3u8"
		case "360p":
			hlsUrls.U360p = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/360p/360p.m3u8"
		case "240p":
			hlsUrls.U240p = baseUrl + "/hls/" + vodBody.Body.Namespace + "/" + vodBody.Body.ID + "/240p/240p.m3u8"
		}
	}

	vodCollection := databse.GetCollection("vod")

	filter := bson.M{"_id": vodBody.MongoId}
	update := bson.M{
		"$set": bson.M{
			"status":    "completed",
			"videoUrls": hlsUrls,
		},
	}

	_, err = vodCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		zap.S().Errorf("failed to update vod: %s", err)
		return err
	}

	zap.S().Infof("completed rtpm hls and updated vod %+v", vodBody)

	return nil
}

func GetAllVods(c *gin.Context) {
	vodCollection := databse.GetCollection("vod")

	filter := bson.D{}
	findOptions := options.Find()

	ctx := c.Request.Context()

	cursor, err := vodCollection.Find(ctx, filter, findOptions)
	if err != nil {
		zap.S().Errorf("failed to get all live streams: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get all live streams"})
		return
	}
	defer cursor.Close(ctx)

	var allVod []bson.M
	if err := cursor.All(ctx, &allVod); err != nil {
		zap.S().Errorf("failed to decode live streams: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode live streams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    allVod,
	})
}
