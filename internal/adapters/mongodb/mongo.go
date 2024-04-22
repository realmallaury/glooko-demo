package mongodb

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
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

func SetUpCollections(ctx context.Context, db *mongo.Database) error {
	err := db.Collection("users").Drop(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to drop users collection")
	}

	userValidation := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"_id", "firstName", "lastName", "dateOfBirth", "phoneNumber", "email"},
			"properties": bson.M{

				"firstName":   bson.M{"bsonType": "string"},
				"lastName":    bson.M{"bsonType": "string"},
				"dateOfBirth": bson.M{"bsonType": "date"},
				"phoneNumber": bson.M{"bsonType": "string"},
				"email":       bson.M{"bsonType": "string"},
				"devices":     bson.M{"bsonType": "array"},
			},
		},
	}

	opt := options.CreateCollection().SetValidator(userValidation)
	if err := db.CreateCollection(ctx, "users", opt); err != nil {
		return errors.Wrap(err, "failed to create collection")
	}

	err = db.Collection("devices").Drop(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to drop devices collection")
	}

	deviceValidation := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"manufacturer", "model", "serialNumber", "userId"},
			"properties": bson.M{
				"manufacturer": bson.M{"bsonType": "string"},
				"model":        bson.M{"bsonType": "string"},
				"serialNumber": bson.M{"bsonType": "string"},
				"userId":       bson.M{"bsonType": "objectId"},
			},
		},
	}

	opt = options.CreateCollection().SetValidator(deviceValidation)
	if err := db.CreateCollection(ctx, "devices", opt); err != nil {
		return errors.Wrap(err, "failed to create collection")
	}

	err = db.Collection("readings").Drop(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to drop reading collection")
	}

	readingValidation := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"userId", "deviceId", "day", "readings"},
			"properties": bson.M{
				"userId":   bson.M{"bsonType": "objectId"},
				"deviceId": bson.M{"bsonType": "objectId"},
				"day":      bson.M{"bsonType": "date"},
				"readings": bson.M{
					"bsonType": "array",
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"time", "value"},
						"properties": bson.M{
							"time":  bson.M{"bsonType": "date"},
							"value": bson.M{"bsonType": "int"},
						},
					},
				},
				"minValue":      bson.M{"bsonType": "int"},
				"maxValue":      bson.M{"bsonType": "int"},
				"avgValue":      bson.M{"bsonType": "double"},
				"sumValues":     bson.M{"bsonType": "int"},
				"countReadings": bson.M{"bsonType": "int"},
			},
		},
	}

	opt = options.CreateCollection().SetValidator(readingValidation)
	if err := db.CreateCollection(ctx, "readings", opt); err != nil {
		return errors.Wrap(err, "failed to create collection")
	}

	readingsIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "deviceId", Value: 1},
			{Key: "day", Value: 1},
			{Key: "readings.time", Value: 1},
		},
	}
	_, err = db.Collection("readings").Indexes().CreateOne(ctx, readingsIndex)
	if err != nil {
		return errors.Wrap(err, "failed to create index")
	}

	return nil
}
