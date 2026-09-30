package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
)

// SysConfigService 系统参数管理。
type SysConfigService struct {
	db *mongo.Database
}

// NewSysConfigService 创建系统参数服务。
func NewSysConfigService(db *mongo.Database) *SysConfigService {
	return &SysConfigService{db: db}
}

// ConfigInput 参数创建/更新。
type ConfigInput struct {
	Key    string `json:"key" binding:"required,min=2,max=64"`
	Name   string `json:"name" binding:"required,max=32"`
	Value  string `json:"value"`
	Remark string `json:"remark"`
}

// List 参数分页。
func (s *SysConfigService) List(ctx context.Context, keyword string, page, size int) ([]*model.SysConfig, int64, error) {
	filter := bson.M{}
	if keyword != "" {
		filter["$or"] = []bson.M{
			{"key": likeFilter(keyword)},
			{"name": likeFilter(keyword)},
		}
	}
	page, size = normalizePage(page, size)
	return pageFind[model.SysConfig](ctx, s.db.Collection(model.ColSysConfig), filter, page, size,
		bson.D{{Key: "createdAt", Value: 1}})
}

// Create 新建参数。
func (s *SysConfigService) Create(ctx context.Context, in *ConfigInput) (*model.SysConfig, error) {
	if n, err := count(ctx, s.db.Collection(model.ColSysConfig), bson.M{"key": in.Key}); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, errs.BadRequest("参数键已存在")
	}
	c := &model.SysConfig{Key: in.Key, Name: in.Name, Value: in.Value, Remark: in.Remark}
	c.PrepareCreate()
	_, err := s.db.Collection(model.ColSysConfig).InsertOne(ctx, c)
	return c, err
}

// Update 更新参数（内置参数仅允许修改值与备注）。
func (s *SysConfigService) Update(ctx context.Context, id string, in *ConfigInput) error {
	var old model.SysConfig
	if err := findOne(ctx, s.db.Collection(model.ColSysConfig), bson.M{"_id": id}, &old); err != nil {
		return errs.NotFound("参数不存在")
	}
	if old.BuiltIn {
		_, err := s.db.Collection(model.ColSysConfig).UpdateOne(ctx, bson.M{"_id": id},
			bson.M{"$set": bson.M{"value": in.Value, "remark": in.Remark, "updatedAt": time.Now()}})
		return err
	}
	if in.Key != old.Key {
		if n, err := count(ctx, s.db.Collection(model.ColSysConfig),
			bson.M{"key": in.Key, "_id": bson.M{"$ne": id}}); err != nil {
			return err
		} else if n > 0 {
			return errs.BadRequest("参数键已存在")
		}
	}
	_, err := s.db.Collection(model.ColSysConfig).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"key": in.Key, "name": in.Name, "value": in.Value,
			"remark": in.Remark, "updatedAt": time.Now()}})
	return err
}

// Delete 删除参数。
func (s *SysConfigService) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的参数")
	}
	if n, err := count(ctx, s.db.Collection(model.ColSysConfig),
		bson.M{"_id": bson.M{"$in": ids}, "builtIn": true}); err != nil {
		return err
	} else if n > 0 {
		return errs.Forbidden("内置参数不允许删除")
	}
	_, err := s.db.Collection(model.ColSysConfig).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}

// Public 返回登录页需要展示的公开信息。
func (s *SysConfigService) Public(ctx context.Context) map[string]string {
	out := map[string]string{"name": "ServerAdmin", "version": "1.0.0", "copyright": ""}
	cursor, err := s.db.Collection(model.ColSysConfig).Find(ctx,
		bson.M{"key": bson.M{"$in": []string{"sys.name", "sys.version", "sys.copyright"}}})
	if err != nil {
		return out
	}
	defer cursor.Close(ctx)
	var list []*model.SysConfig
	if cursor.All(ctx, &list) == nil {
		for _, c := range list {
			switch c.Key {
			case "sys.name":
				if c.Value != "" {
					out["name"] = c.Value
				}
			case "sys.version":
				if c.Value != "" {
					out["version"] = c.Value
				}
			case "sys.copyright":
				out["copyright"] = c.Value
			}
		}
	}
	return out
}
