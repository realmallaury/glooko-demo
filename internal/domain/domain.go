package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a person in the system.
type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	FirstName   string             `bson:"firstName"`
	LastName    string             `bson:"lastName"`
	DateOfBirth time.Time          `bson:"dateOfBirth"`
	Email       string             `bson:"email"`
	PhoneNumber string             `bson:"phoneNumber"`
	Devices     []Device           `bson:"devices"` // Embedding devices for simplicity, might reference by IDs in a real app.
}

// Device represents a glucose measuring device used by a user.
type Device struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	UserID       primitive.ObjectID `bson:"userId"`
	Manufacturer string             `bson:"manufacturer"`
	Model        string             `bson:"model"`
	SerialNumber string             `bson:"serialNumber"`
}

// Reading represents a glucose level reading taken from a device.
type ReadingEntry struct {
	Time  time.Time `bson:"time"`
	Value int       `bson:"value"`
}

// Reading represents a glucose level readings for a day taken from a device.
type Reading struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"userId"`
	DeviceID      primitive.ObjectID `bson:"deviceId"`
	Day           time.Time          `bson:"day"`
	Readings      []ReadingEntry     `bson:"readings"`
	MinValue      int                `bson:"minValue"`
	MaxValue      int                `bson:"maxValue"`
	AvgValue      float64            `bson:"avgValue"`
	SumValues     int                `bson:"sumValues"`
	CountReadings int                `bson:"countReadings"`
}
