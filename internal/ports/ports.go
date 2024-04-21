package ports

import (
	"context"
	"glooko/internal/domain"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRepository interface {
	Save(ctx context.Context, user domain.User) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
}

type DeviceRepository interface {
	Save(ctx context.Context, device domain.Device) (domain.Device, error)
	FindByID(ctx context.Context, id string) (domain.Device, error)
}

type ReadingRepository interface {
	AddReadingAndUpdateStats(ctx context.Context, deviceID, userID primitive.ObjectID, value int, timestamp time.Time) error
	FetchReadings(ctx context.Context, userID string, startDate, endDate time.Time) ([]domain.Reading, error)
}
