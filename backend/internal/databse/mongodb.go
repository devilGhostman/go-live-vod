package databse

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

var client *mongo.Client
var database *mongo.Database

func ConnectDB(uri string, dbName string) {
	var err error
	client, err = mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		zap.S().Errorf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		zap.S().Errorf("Failed to ping MongoDB: %v", err)
	}

	database = client.Database(dbName)
	zap.S().Infof("Connected to MongoDB at %s", uri)
}

func DisconnectDB() {
	if err := client.Disconnect(context.Background()); err != nil {
		zap.S().Errorf("Failed to disconnect from MongoDB: %v", err)
	}
	zap.S().Info("Disconnected from MongoDB")
}

func GetCollection(collectionName string) *mongo.Collection {
	return database.Collection(collectionName)
}
