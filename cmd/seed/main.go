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
			for day := 0; day < daysInPast; day++ {
				date := startDate.AddDate(0, 0, day)
				for j := 0; j < 24*12; j++ { // Generate readings every 5 minutes
					err = readingRepo.AddReadingAndUpdateStats(ctx, d.ID.Hex(), u.ID.Hex(), rand.Intn(1024), date.Add(5*time.Minute*time.Duration(j)))
					if err != nil {
						log.Fatal("Failed to add reading", zap.Error(err))
					}
				}
			}
		}
	}

	log.Infof("Seeded %d users with devices and readings", len(users))
}
