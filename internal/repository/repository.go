// Package repository 数据访问层：所有对 MongoDB 的读写都集中在这里。
// Service 只负责业务规则与缓存编排，不直接接触 mongo.Collection。
package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repos 聚合所有数据访问仓库，统一构建与传递。
type Repos struct {
	User       *UserRepository
	Role       *RoleRepository
	Menu       *MenuRepository
	Department *DepartmentRepository
	Dict       *DictRepository
	SysConfig  *SysConfigRepository
	Log        *LogRepository
	File       *FileRepository
	Notice     *NoticeRepository
	Job        *JobRepository
	Token      *TokenRepository
	Dashboard  *DashboardRepository
}

// NewAll 构建全部仓库。
func NewAll(db *mongo.Database) *Repos {
	return &Repos{
		User:       NewUserRepository(db),
		Role:       NewRoleRepository(db),
		Menu:       NewMenuRepository(db),
		Department: NewDepartmentRepository(db),
		Dict:       NewDictRepository(db),
		SysConfig:  NewSysConfigRepository(db),
		Log:        NewLogRepository(db),
		File:       NewFileRepository(db),
		Notice:     NewNoticeRepository(db),
		Job:        NewJobRepository(db),
		Token:      NewTokenRepository(db),
		Dashboard:  NewDashboardRepository(db),
	}
}

// collection 泛型数据访问基类：封装游标遍历与 CRUD 样板代码。
// 各领域仓库内嵌它获得标准数据访问能力，并提供少量语义化方法。
type collection[T any] struct {
	coll *mongo.Collection
}

func collectionOf[T any](db *mongo.Database, name string) collection[T] {
	return collection[T]{coll: db.Collection(name)}
}

// FindOne 按过滤条件取单条，未命中返回 mongo.ErrNoDocuments。
func (c collection[T]) FindOne(ctx context.Context, filter bson.M) (*T, error) {
	var doc T
	if err := c.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// FindAll 按过滤条件取全部；sort 为 nil 时按创建时间倒序。
func (c collection[T]) FindAll(ctx context.Context, filter bson.M, sort bson.D) ([]*T, error) {
	opts := options.Find()
	if sort == nil {
		sort = bson.D{{Key: "createdAt", Value: -1}}
	}
	opts.SetSort(sort)
	cursor, err := c.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]*T, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// Page 分页查询；sort 为 nil 时按创建时间倒序。
func (c collection[T]) Page(ctx context.Context, filter bson.M, page, size int, sort bson.D) ([]*T, int64, error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*T{}, 0, nil
	}
	if sort == nil {
		sort = bson.D{{Key: "createdAt", Value: -1}}
	}
	opts := options.Find().
		SetSort(sort).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cursor, err := c.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	list := make([]*T, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Count 计数。
func (c collection[T]) Count(ctx context.Context, filter bson.M) (int64, error) {
	return c.coll.CountDocuments(ctx, filter)
}

// Insert 插入一条文档。
func (c collection[T]) Insert(ctx context.Context, doc *T) error {
	_, err := c.coll.InsertOne(ctx, doc)
	return err
}

// UpdateSet 按过滤条件执行 $set 更新。
func (c collection[T]) UpdateSet(ctx context.Context, filter, set bson.M) error {
	_, err := c.coll.UpdateOne(ctx, filter, bson.M{"$set": set})
	return err
}

// Update 按过滤条件执行任意更新（$inc、$set 组合等）。
func (c collection[T]) Update(ctx context.Context, filter, update bson.M) error {
	_, err := c.coll.UpdateOne(ctx, filter, update)
	return err
}

// Delete 按过滤条件删除，返回删除数量。
func (c collection[T]) Delete(ctx context.Context, filter bson.M) (int64, error) {
	res, err := c.coll.DeleteMany(ctx, filter)
	return res.DeletedCount, err
}

// DeleteByIDs 按主键批量删除，空列表直接返回 0。
func (c collection[T]) DeleteByIDs(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	return c.Delete(ctx, bson.M{"_id": bson.M{"$in": ids}})
}

// FindByIDs 按主键批量查询。
func (c collection[T]) FindByIDs(ctx context.Context, ids []string) ([]*T, error) {
	if len(ids) == 0 {
		return []*T{}, nil
	}
	return c.FindAll(ctx, bson.M{"_id": bson.M{"$in": ids}}, nil)
}
