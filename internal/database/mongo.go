// Package database 负责 MongoDB 连接与索引初始化。
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"serveradmin/internal/config"
	"serveradmin/internal/model"
)

// Connect 连接 MongoDB 并返回 Database。
func Connect(cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	timeout := time.Duration(cfg.Mongo.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(cfg.Mongo.URI).
		SetConnectTimeout(timeout).
		SetServerSelectionTimeout(timeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("连接 MongoDB 失败: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("MongoDB Ping 失败（请确认服务已启动，URI: %s）: %w", cfg.Mongo.URI, err)
	}
	return client, client.Database(cfg.Mongo.Database), nil
}

// EnsureIndexes 创建全部业务索引（幂等）。
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	models := map[string][]mongo.IndexModel{
		model.ColUser: {
			{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "deptId", Value: 1}}},
		},
		model.ColRole: {
			{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		model.ColMenu: {
			{Keys: bson.D{{Key: "parentId", Value: 1}}},
		},
		model.ColDepartment: {
			{Keys: bson.D{{Key: "parentId", Value: 1}}},
		},
		model.ColDictType: {
			{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		model.ColDictItem: {
			{Keys: bson.D{{Key: "typeCode", Value: 1}, {Key: "value", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		model.ColSysConfig: {
			{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		model.ColOperationLog: {
			{Keys: bson.D{{Key: "createdAt", Value: -1}}},
			{Keys: bson.D{{Key: "username", Value: 1}}},
		},
		model.ColLoginLog: {
			{Keys: bson.D{{Key: "loginAt", Value: -1}}},
			{Keys: bson.D{{Key: "username", Value: 1}}},
		},
		model.ColNotice: {
			{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		},
		model.ColFile: {
			{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		},
		model.ColJob: {
			{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		model.ColJobLog: {
			{Keys: bson.D{{Key: "jobName", Value: 1}, {Key: "runAt", Value: -1}}},
		},
		model.ColTokenBlacklist: {
			// TTL 索引：到达 expireAt 后自动删除
			{Keys: bson.D{{Key: "expireAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
	}

	for collName, idxModels := range models {
		if _, err := db.Collection(collName).Indexes().CreateMany(ctx, idxModels); err != nil {
			return fmt.Errorf("为集合 %s 创建索引失败: %w", collName, err)
		}
	}
	return nil
}
