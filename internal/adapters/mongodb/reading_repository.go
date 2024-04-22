package mongodb

import (
	"context"
	"glooko/internal/domain"
	"time"

	"github.com/pkg/errors"
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

func (r *ReadingRepository) AddReadingAndUpdateStats(ctx context.Context, deviceID, userID string, value int, timestamp time.Time) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.Wrap(err, "failed to parse userID")
	}

	deviceObjID, err := primitive.ObjectIDFromHex(deviceID)
	if err != nil {
		return errors.Wrap(err, "failed to parse deviceID")
	}

	day := timestamp.Truncate(24 * time.Hour)

	filter := bson.M{"deviceId": deviceID, "day": day}
	update := bson.D{
		{Key: "$push", Value: bson.M{"readings": bson.M{"time": timestamp, "value": value}}},
		{Key: "$min", Value: bson.M{"minValue": value}},
		{Key: "$max", Value: bson.M{"maxValue": value}},
		{Key: "$inc", Value: bson.M{"sumValues": value, "countReadings": 1}},
		{Key: "$setOnInsert", Value: bson.M{"userId": userObjectID, "day": day, "deviceId": deviceObjID}},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var updatedReading domain.Reading

	// Add new reading and update min/max
	err = r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedReading)
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
		return nil, errors.Wrap(err, "failed to parse userID")
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
		return nil, errors.Wrap(err, "failed to find readings")
	}
	defer cursor.Close(ctx)

	var readings []domain.Reading
	if err = cursor.All(ctx, &readings); err != nil {
		return nil, errors.Wrap(err, "failed to decode readings")
	}

	return readings, nil
}

func (r *ReadingRepository) FetchDevicesOverview(ctx context.Context, userID string, days int) ([]domain.DayDeviceCounts, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse userID")
	}

	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"userId": userObjectID, "day": bson.M{"$gte": startDate}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"day": "$day", "deviceId": "$deviceId"},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":     "$_id.day",
			"devices": bson.M{"$push": bson.M{"deviceId": "$_id.deviceId", "count": "$count"}},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":     0,
			"day":     "$_id",
			"devices": 1,
		}}},
		{{Key: "$sort", Value: bson.M{"day": 1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.DayDeviceCounts
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
