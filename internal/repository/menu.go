package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// MenuRepository 菜单集合数据访问。
type MenuRepository struct {
	collection[model.Menu]
}

// NewMenuRepository 创建菜单仓库。
func NewMenuRepository(db *mongo.Database) *MenuRepository {
	return &MenuRepository{collectionOf[model.Menu](db, model.ColMenu)}
}

// FindDescendantIDs 广度优先收集全部子孙菜单 ID。
func (r *MenuRepository) FindDescendantIDs(ctx context.Context, roots []string) ([]string, error) {
	result := make([]string, 0)
	frontier := roots
	for len(frontier) > 0 {
		children, err := r.FindAll(ctx, bson.M{"parentId": bson.M{"$in": frontier}}, nil)
		if err != nil {
			return nil, err
		}
		frontier = frontier[:0]
		for _, c := range children {
			result = append(result, c.ID)
			frontier = append(frontier, c.ID)
		}
	}
	return result, nil
}
