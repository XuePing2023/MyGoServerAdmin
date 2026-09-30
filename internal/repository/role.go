package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// RoleRepository 角色集合数据访问。
type RoleRepository struct {
	collection[model.Role]
}

// NewRoleRepository 创建角色仓库。
func NewRoleRepository(db *mongo.Database) *RoleRepository {
	return &RoleRepository{collectionOf[model.Role](db, model.ColRole)}
}

// FindByCodes 按角色编码查询。
func (r *RoleRepository) FindByCodes(ctx context.Context, codes []string) ([]*model.Role, error) {
	if len(codes) == 0 {
		return []*model.Role{}, nil
	}
	return r.FindAll(ctx, bson.M{"code": bson.M{"$in": codes}}, nil)
}

// CountUsersByRoleCodes 统计正在使用这些角色的用户数。
func (r *RoleRepository) CountUsersByRoleCodes(ctx context.Context, codes []string) (int64, error) {
	return r.coll.Database().Collection(model.ColUser).
		CountDocuments(ctx, bson.M{"roles": bson.M{"$in": codes}})
}
