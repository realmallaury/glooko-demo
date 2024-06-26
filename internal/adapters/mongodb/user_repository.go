package mongodb

import (
	"context"
	"glooko/internal/domain"
	"glooko/internal/ports"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (r *UserRepository) Save(ctx context.Context, user domain.User) (domain.User, error) {
	result, err := r.db.InsertOne(ctx, user)
	if err != nil {
		return domain.User{}, errors.Wrap(err, "failed to save user")
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return domain.User{}, errors.Errorf("failed to convert inserted id to ObjectID")
	}

	var savedUser domain.User
	err = r.db.FindOne(ctx, bson.M{"_id": oid}).Decode(&savedUser)
	if err != nil {
		return domain.User{}, errors.Wrap(err, "failed to fetch saved user")
	}

	return savedUser, nil
}
