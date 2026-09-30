package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// UserRepository 用户集合数据访问。
type UserRepository struct {
	collection[model.User]
}

// NewUserRepository 创建用户仓库。
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{collectionOf[model.User](db, model.ColUser)}
}

// FindByUsername 按用户名取用户（登录用）。
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.FindOne(ctx, bson.M{"username": username})
}
