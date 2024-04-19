package domain

import "time"

// User represents a person in the system.
type User struct {
	ID          string    `bson:"_id"`
	FirstName   string    `bson:"first_name"`
	LastName    string    `bson:"last_name"`
	DateOfBirth time.Time `bson:"date_of_birth"`
	Email       string    `bson:"email"`
	PhoneNumber string    `bson:"phone_number"`
	Devices     []Device  `bson:"devices"` // Embedding devices for simplicity, might reference by IDs in a real app.
}

// Device represents a glucose measuring device used by a user.
type Device struct {
	ID           string `bson:"_id"`
	Manufacturer string `bson:"manufacturer"`
	Model        string `bson:"model"`
	SerialNumber string `bson:"serial_number"`
	Type         string `bson:"type"` // Could be "BG" for blood glucose reader or "CGM" for continuous glucose monitor.
}

// Reading represents a glucose level reading taken from a device.
type Reading struct {
	ID           string    `bson:"_id"`
	DeviceID     string    `bson:"device_id"`
	UserID       string    `bson:"user_id"`
	Timestamp    time.Time `bson:"timestamp"`
	GlucoseValue int       `bson:"glucose_value"`
}
