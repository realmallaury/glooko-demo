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
	readingRepo := mongodb.NewReadingRepository(mongoDB)

	fmt.Println("days\tno\t\tFR-FD")

	for daysInPast := 90; daysInPast <= 120; daysInPast++ {
		noOfReadings := 0
		var testUser primitive.ObjectID
		readingsBatch := []domain.ReadingEntry{}

		for userCount := 1; userCount <= daysInPast*2; userCount++ {
			user := domain.User{
				FirstName:   "John" + strconv.Itoa(userCount),
				LastName:    "Doe",
				DateOfBirth: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
				Email:       fmt.Sprintf("john.doe%d@example.com", userCount),
				PhoneNumber: fmt.Sprintf("555-01%02d", userCount),
				Devices:     []domain.Device{},
			}
			u, err := userRepo.Save(ctx, user)
			if err != nil {
				log.Fatal("failed to save user", zap.Error(err))
			}

			device := domain.Device{
				Manufacturer: "Acme",
				Model:        "X100",
				SerialNumber: "SN00" + strconv.Itoa(userCount),
				UserID:       u.ID,
			}
			d, err := deviceRepo.Save(ctx, device)
			if err != nil {
				log.Fatal("Failed to save device", zap.Error(err))
			}

			testUser = u.ID // This will always be the last user added

			startDate := time.Now().AddDate(0, 0, -daysInPast)
			for day := 0; day < daysInPast; day++ {
				date := startDate.AddDate(0, 0, day)
				for j := 0; j < 24*12; j++ { // Generate readings every 5 minutes
					noOfReadings++
					readingEntry := domain.ReadingEntry{
						Time:  date.Add(5 * time.Minute * time.Duration(j)),
						Value: rand.Intn(1024),
					}
					readingsBatch = append(readingsBatch, readingEntry)
				}
			}

			err = addDayReadingsAndUpdateStats(ctx, mongoDB.Database, d.ID.Hex(), u.ID.Hex(), startDate, readingsBatch)
			if err != nil {
				log.Fatal("Failed to add daily batch of readings", zap.Error(err))
			}
			readingsBatch = []domain.ReadingEntry{} // Reset for the next batch
		}

		// Measure performance with the current number of users and days
		startTime := time.Now()
		_, err = readingRepo.FetchReadings(ctx, testUser.Hex(), time.Now().AddDate(0, 0, -daysInPast), time.Now()) // Modify accordingly
		readingsTime := time.Since(startTime)
		if err != nil {
			log.Fatal("Failed to fetch readings", zap.Error(err))
		}

		startTime = time.Now()
		_, err = readingRepo.FetchDevicesOverview(ctx, testUser.Hex(), daysInPast) // Modify accordingly
		devicesOverviewTime := time.Since(startTime)
		if err != nil {
			log.Fatal("Failed to fetch device overview", zap.Error(err))
		}

		fmt.Printf("%d\t%d\t\t%s-%s\n", daysInPast, noOfReadings, readingsTime, devicesOverviewTime)
	}
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
