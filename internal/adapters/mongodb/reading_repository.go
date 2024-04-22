package mongodb

import (
	"context"
	"fmt"
	"glooko/internal/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReadingRepository struct {
	collection *mongo.Collection
}

func NewReadingRepository(db *MongoDB) *ReadingRepository {
	return &ReadingRepository{
		collection: db.Database.Collection("readings"),
	}
}

func (r *ReadingRepository) AddReadingAndUpdateStats(ctx context.Context, deviceID, userID primitive.ObjectID, value int, timestamp time.Time) error {
	day := timestamp.Truncate(24 * time.Hour)

	filter := bson.M{"deviceId": deviceID, "day": day}
	update := bson.D{
		{Key: "$push", Value: bson.M{"readings": bson.M{"time": timestamp, "value": value}}},
		{Key: "$min", Value: bson.M{"minValue": value}},
		{Key: "$max", Value: bson.M{"maxValue": value}},
		{Key: "$inc", Value: bson.M{"sumValues": value, "countReadings": 1}},
		{Key: "$setOnInsert", Value: bson.M{"userId": userID, "day": day, "deviceId": deviceID}},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var updatedReading domain.Reading

	// Add new reading and update min/max
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedReading)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	if updatedReading.CountReadings > 0 {
		newAvg := float64(updatedReading.SumValues) / float64(updatedReading.CountReadings)
		statsUpdate := bson.M{"$set": bson.M{"avgValue": newAvg}}
		_, err = r.collection.UpdateOne(ctx, filter, statsUpdate)
	}

	return err
}

func (r *ReadingRepository) FetchReadings(ctx context.Context, userID string, startDate, endDate time.Time) ([]domain.Reading, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse userID: %v", err)
	}

	filter := bson.M{
		"userId": userObjectID,
		"day": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "day", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find readings: %v", err)
	}
	defer cursor.Close(ctx)

	var readings []domain.Reading
	if err = cursor.All(ctx, &readings); err != nil {
		return nil, fmt.Errorf("failed to parse readings: %v", err)
	}

	return readings, nil
}
