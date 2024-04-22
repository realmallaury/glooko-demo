package ports

import (
	"context"
	"glooko/internal/domain"
	"time"
)

type UserRepository interface {
	Save(ctx context.Context, user domain.User) (domain.User, error)
}

type DeviceRepository interface {
	Save(ctx context.Context, device domain.Device) (domain.Device, error)
}

type ReadingRepository interface {
	AddReadingAndUpdateStats(ctx context.Context, deviceID, userID string, value int, timestamp time.Time) error
	FetchReadings(ctx context.Context, userID string, startDate, endDate time.Time) ([]domain.Reading, error)
	FetchDevicesOverview(ctx context.Context, userID string, days int) ([]domain.DayDeviceCounts, error)
}
