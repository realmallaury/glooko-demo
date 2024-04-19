package mongodb

import (
	"context"
	"glooko/internal/domain"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const DevicesCollection = "devices"

type DeviceRepository struct {
	collection *mongo.Collection
}

func NewDeviceRepository(db *MongoDB) *DeviceRepository {
	return &DeviceRepository{
		collection: db.Database.Collection(DevicesCollection),
	}
}

func (r *DeviceRepository) Save(ctx context.Context, device domain.Device) error {
	_, err := r.collection.InsertOne(ctx, device)
	if err != nil {
		return errors.Wrap(err, "failed to save device")
	}

	return nil
}

func (r *DeviceRepository) FindByID(ctx context.Context, id string) (domain.Device, error) {
	var device domain.Device
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&device)
	if err != nil {
		return domain.Device{}, errors.Wrap(err, "failed to find device")
	}

	return device, err
}
