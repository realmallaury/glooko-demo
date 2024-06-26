package main

import (
	"context"
	"fmt"
	"glooko/internal/adapters/mongodb"
	"glooko/internal/config"
	"glooko/internal/domain"
	"math/rand"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	log := logger.Sugar()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	mongoDB, err := mongodb.NewMongoDB(ctx, cfg.MongoDBURI, cfg.MongoDBName)
	if err != nil {
		log.Fatal("failed to connect to MongoDB", zap.Error(err))
	}

	err = mongodb.SetUpCollections(ctx, mongoDB.Database)
	if err != nil {
		log.Fatal("failed to set up collections", zap.Error(err))
	}

	userRepo := mongodb.NewUserRepository(mongoDB)
	deviceRepo := mongodb.NewDeviceRepository(mongoDB)

	users := make([]domain.User, 10)
	devices := []domain.Device{
		{Manufacturer: "Acme", Model: "X100", SerialNumber: "SN0001"},
		{Manufacturer: "Acme", Model: "X200", SerialNumber: "SN0002"},
		{Manufacturer: "Beta", Model: "Y100", SerialNumber: "SN0003"},
		{Manufacturer: "Beta", Model: "Y200", SerialNumber: "SN0004"},
		{Manufacturer: "Gamma", Model: "Z100", SerialNumber: "SN0005"},
	}

	for i := range users {
		users[i] = domain.User{
			FirstName:   "FirstName" + strconv.Itoa(i),
			LastName:    "LastName" + strconv.Itoa(i),
			DateOfBirth: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			Email:       fmt.Sprintf("user%d@example.com", i),
			PhoneNumber: fmt.Sprintf("555-010%d", i),
			Devices:     []domain.Device{},
		}
		u, err := userRepo.Save(ctx, users[i])
		if err != nil {
			log.Fatal("failed to save user", zap.Error(err))
		}

		numDevices := rand.Intn(3) + 1
		for j := 0; j < numDevices; j++ {
			deviceIndex := rand.Intn(len(devices))
			device := devices[deviceIndex]

			device.UserID = u.ID
			d, err := deviceRepo.Save(ctx, device)
			if err != nil {
				log.Fatal("Failed to save device", zap.Error(err))
			}

			daysInPast := 5
			startDate := time.Now().AddDate(0, 0, -daysInPast)
			readingsBatch := []domain.ReadingEntry{}

			for day := 0; day < daysInPast; day++ {
				date := startDate.AddDate(0, 0, day)
				for j := 0; j < 24*12; j++ { // Generate readings every 5 minutes
					reading := domain.ReadingEntry{
						Time:  date.Add(5 * time.Minute * time.Duration(j)),
						Value: rand.Intn(1024),
					}
					readingsBatch = append(readingsBatch, reading)
				}
				// Process the batch for each day
				err = addDayReadingsAndUpdateStats(ctx, mongoDB.Database, d.ID.Hex(), u.ID.Hex(), date, readingsBatch)
				if err != nil {
					log.Fatal("Failed to add day batch of readings", zap.Error(err))
				}
				readingsBatch = []domain.ReadingEntry{} // Clear the batch for the next day
			}
		}
	}

	log.Infof("Seeded %d users with devices and readings", len(users))
}

func addDayReadingsAndUpdateStats(ctx context.Context, db *mongo.Database, deviceID, userID string, day time.Time, readings []domain.ReadingEntry) error {
	col := db.Collection("readings")

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.Wrap(err, "failed to parse userID")
	}

	deviceObjID, err := primitive.ObjectIDFromHex(deviceID)
	if err != nil {
		return errors.Wrap(err, "failed to parse deviceID")
	}

	// Normalize day to ensure consistent day boundary
	normalizedDay := day.Truncate(24 * time.Hour)
	filter := bson.M{"deviceId": deviceObjID, "day": normalizedDay}
	update := bson.D{{Key: "$setOnInsert", Value: bson.M{"userId": userObjectID, "deviceId": deviceObjID, "day": normalizedDay}}}

	readingsArray := bson.A{}
	sumValues := 0
	minValue := int(^uint(0) >> 1) // Initialize with the maximum int value
	maxValue := -1

	for _, reading := range readings {
		readingsArray = append(readingsArray, bson.M{"time": reading.Time, "value": reading.Value})
		sumValues += reading.Value
		if reading.Value < minValue {
			minValue = reading.Value
		}
		if reading.Value > maxValue {
			maxValue = reading.Value
		}
	}

	update = append(update, bson.E{Key: "$push", Value: bson.M{"readings": bson.M{"$each": readingsArray}}})
	update = append(update, bson.E{Key: "$min", Value: bson.M{"minValue": minValue}})
	update = append(update, bson.E{Key: "$max", Value: bson.M{"maxValue": maxValue}})
	update = append(update, bson.E{Key: "$inc", Value: bson.M{"sumValues": sumValues, "countReadings": len(readings)}})

	opts := options.FindOneAndUpdate().SetUpsert(true)
	var updatedReading domain.Reading

	// Perform the update and calculate the new average if needed
	err = col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedReading)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	if updatedReading.CountReadings > 0 {
		newAvg := float64(updatedReading.SumValues) / float64(updatedReading.CountReadings)
		avgUpdate := bson.M{"$set": bson.M{"avgValue": newAvg}}
		_, err = col.UpdateOne(ctx, filter, avgUpdate)
		if err != nil {
			return err
		}
	}

	return nil
}
