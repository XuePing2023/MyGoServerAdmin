package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/cache"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/repository"
)

// SysConfigService 系统参数管理。
type SysConfigService struct {
	repo  *repository.SysConfigRepository
	cache *cache.Helper
}

// NewSysConfigService 创建系统参数服务。
func NewSysConfigService(repo *repository.SysConfigRepository, helper *cache.Helper) *SysConfigService {
	return &SysConfigService{repo: repo, cache: helper}
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
	return s.repo.Page(ctx, filter, page, size,
		bson.D{{Key: "createdAt", Value: 1}})
}

// Create 新建参数。
func (s *SysConfigService) Create(ctx context.Context, in *ConfigInput) (*model.SysConfig, error) {
	if n, err := s.repo.Count(ctx, bson.M{"key": in.Key}); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, errs.BadRequest("参数键已存在")
	}
	c := &model.SysConfig{Key: in.Key, Name: in.Name, Value: in.Value, Remark: in.Remark}
	c.PrepareCreate()
	if err := s.repo.Insert(ctx, c); err != nil {
		return nil, err
	}
	// 写操作成功后失效 PUBLIC 缓存
	s.cache.Invalidate(ctx, cache.BizPrefix("system"))
	return c, nil
}

// Update 更新参数（内置参数仅允许修改值与备注）。
func (s *SysConfigService) Update(ctx context.Context, id string, in *ConfigInput) error {
	old, err := s.repo.FindOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errs.NotFound("参数不存在")
	}
	if old.BuiltIn {
		if err := s.repo.UpdateSet(ctx, bson.M{"_id": id},
			bson.M{"value": in.Value, "remark": in.Remark, "updatedAt": time.Now()}); err != nil {
			return err
		}
		s.cache.Invalidate(ctx, cache.BizPrefix("system"))
		return nil
	}
	if in.Key != old.Key {
		if n, err := s.repo.Count(ctx,
			bson.M{"key": in.Key, "_id": bson.M{"$ne": id}}); err != nil {
			return err
		} else if n > 0 {
			return errs.BadRequest("参数键已存在")
		}
	}
	if err := s.repo.UpdateSet(ctx, bson.M{"_id": id},
		bson.M{"key": in.Key, "name": in.Name, "value": in.Value,
			"remark": in.Remark, "updatedAt": time.Now()}); err != nil {
		return err
	}
	s.cache.Invalidate(ctx, cache.BizPrefix("system"))
	return nil
}

// Delete 删除参数。
func (s *SysConfigService) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的参数")
	}
	if n, err := s.repo.Count(ctx,
		bson.M{"_id": bson.M{"$in": ids}, "builtIn": true}); err != nil {
		return err
	} else if n > 0 {
		return errs.Forbidden("内置参数不允许删除")
	}
	if _, err := s.repo.DeleteByIDs(ctx, ids); err != nil {
		return err
	}
	s.cache.Invalidate(ctx, cache.BizPrefix("system"))
	return nil
}

// Public 返回登录页需要展示的公开信息（PUBLIC 策略：所有用户相同，
// Key: serveradmin:v1:system:config）。
func (s *SysConfigService) Public(ctx context.Context) (map[string]string, error) {
	return cache.GetJSON(s.cache, ctx, cache.PublicKey("system", "config"), s.cache.TTL.Public,
		func(ctx context.Context) (map[string]string, error) {
			out := map[string]string{"name": "ServerAdmin", "version": "1.0.0", "copyright": ""}
			list, err := s.repo.FindByKeys(ctx,
				[]string{"sys.name", "sys.version", "sys.copyright"})
			if err != nil {
				return nil, err
			}
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
			return out, nil
		})
}
