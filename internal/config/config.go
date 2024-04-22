package config

import (
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	MongoDBURI  string `validate:"required"`
	MongoDBName string `validate:"required"`
	ServerPort  string `validate:"required"`
}

func LoadConfig() (*Config, error) {
	config := &Config{
		MongoDBURI:  os.Getenv("MONGODB_URI"),
		MongoDBName: os.Getenv("MONGODB_NAME"),
		ServerPort:  os.Getenv("SERVER_PORT"),
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, errors.Wrap(err, "failed to validate config")
	}

	return config, nil
}
