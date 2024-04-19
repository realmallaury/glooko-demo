package ports

import (
	"context"
	"glooko/internal/domain"
)

type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	FindByID(ctx context.Context, id string) (domain.User, error)
}

type DeviceRepository interface {
	Save(ctx context.Context, device domain.Device) error
	FindByID(ctx context.Context, id string) (domain.Device, error)
}
