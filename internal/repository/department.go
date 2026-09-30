package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// DepartmentRepository 部门集合数据访问。
type DepartmentRepository struct {
	collection[model.Department]
}

// NewDepartmentRepository 创建部门仓库。
func NewDepartmentRepository(db *mongo.Database) *DepartmentRepository {
	return &DepartmentRepository{collectionOf[model.Department](db, model.ColDepartment)}
}

// SelfAndDescendantIDs 返回自身及全部子孙部门 ID。
func (r *DepartmentRepository) SelfAndDescendantIDs(ctx context.Context, root string) ([]string, error) {
	result := []string{root}
	frontier := []string{root}
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

// CountUsers 统计部门下的用户数。
func (r *DepartmentRepository) CountUsers(ctx context.Context, deptID string) (int64, error) {
	return r.coll.Database().Collection(model.ColUser).
		CountDocuments(ctx, bson.M{"deptId": deptID})
}
