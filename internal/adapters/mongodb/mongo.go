package mongodb

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Database *mongo.Database
}

func NewMongoDB(ctx context.Context, uri string, dbName string) (*MongoDB, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to MongoDB")
	}

	db := client.Database(dbName)
	return &MongoDB{Database: db}, nil
}
