package main

import (
	"flag"
	rabbitmq "github/devilGhostman/backend/internal/rabbitMq"
	"github/devilGhostman/backend/internal/databse"
	"github/devilGhostman/backend/internal/package/logger"
	"github/devilGhostman/backend/internal/package/shutdown"
	"github/devilGhostman/backend/internal/server"
	videoOnDemand "github/devilGhostman/backend/internal/services/vod"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// setting up logger
	zaplogger := logger.SetupGlobalLogger()
	defer zaplogger.Sync()

	// setting up env
	err := godotenv.Load()
	if err != nil {
		zap.S().Error("can not load env")
	}

	addr := flag.String("address", "0.0.0.0:8000", "address of server you want to run on")

	// setting up server
	httpServer := server.NewServer(*addr)
	httpServer.SetupRouter()

	// setting up graceful shutdown
	shutdown.EnableGrafullyShutdown()

	//seeting up mongodb
	mongoUri := os.Getenv("MONGO_URI")
	databse.ConnectDB(mongoUri, "go-transcode")
	defer databse.DisconnectDB()

	// setting up rabbitmq
	rabbitMqUri := os.Getenv("RABBITMQ_URL")
	rabbitmqQueue := os.Getenv("RABBITMQ_QUEUE")
	rabbitmqExchange := os.Getenv("RABBITMQ_EXCHANGE")

	rabbitMq := rabbitmq.NewRabbitMq(
		rabbitMqUri,
		rabbitmqQueue,
		rabbitmqExchange,
	)
	rabbitMq.SetupRabbitMq()

	mss, err := rabbitMq.ConsumeMessages(rabbitMq.Channel)
	if err != nil {
		zap.S().Errorf("failed to consume messages: %s", err)
	}

	go func() {
		for msg := range mss {
			zap.S().Infof("Received a message: %s", msg.Body)
			err := videoOnDemand.StartVodTranscoding(msg.Body)

			if err != nil {
				zap.S().Errorf("failed to start vod transcoding: %s", err)
				msg.Nack(false, true)
			}
			msg.Ack(true)
		}
	}()

	// running server
	httpServer.Run()

}
