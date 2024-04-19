package mongodb

import (
	"context"
	"glooko/internal/domain"
	"glooko/internal/ports"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const UsersCollection = "users"

type UserRepository struct {
	db *mongo.Collection
}

func NewUserRepository(db *MongoDB) ports.UserRepository {
	return &UserRepository{
		db: db.Database.Collection(UsersCollection),
	}
}

func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	_, err := r.db.InsertOne(ctx, user)
	if err != nil {
		return errors.Wrap(err, "failed to save user")
	}

	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	var user domain.User
	err := r.db.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return domain.User{}, errors.Wrap(err, "failed to find user")
	}

	return user, err
}
