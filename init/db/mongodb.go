package db

import (
	"context"
	"fmt"
	"github.com/vlasdash/redditclone/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongo() (*mongo.Database, error) {
	host := config.C.Mongo.Host
	port := config.C.Mongo.Port
	dbName := config.C.Mongo.Name

	url := fmt.Sprintf("mongodb://%s:%d/", host, port)
	option := options.Client().ApplyURI(url)
	client, err := mongo.Connect(context.TODO(), option)
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	return db, nil
}
