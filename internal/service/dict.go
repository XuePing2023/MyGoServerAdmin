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

// DictService 字典管理。
type DictService struct {
	repo  *repository.DictRepository
	cache *cache.Helper
}

// NewDictService 创建字典服务。
func NewDictService(repo *repository.DictRepository, helper *cache.Helper) *DictService {
	return &DictService{repo: repo, cache: helper}
}

// DictTypeInput 字典类型。
type DictTypeInput struct {
	Name   string `json:"name" binding:"required,max=32"`
	Code   string `json:"code" binding:"required,min=2,max=32"`
	Status int    `json:"status" binding:"required,oneof=1 2"`
	Remark string `json:"remark"`
}

// DictItemInput 字典项。
type DictItemInput struct {
	TypeCode string `json:"typeCode" binding:"required"`
	Label    string `json:"label" binding:"required,max=32"`
	Value    string `json:"value" binding:"required,max=64"`
	TagType  string `json:"tagType" binding:"omitempty,oneof=success info warning danger default"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status" binding:"required,oneof=1 2"`
	Remark   string `json:"remark"`
}

// TypeList 字典类型分页。
func (s *DictService) TypeList(ctx context.Context, name string, page, size int) ([]*model.DictType, int64, error) {
	filter := bson.M{}
	if name != "" {
		filter["name"] = likeFilter(name)
	}
	page, size = normalizePage(page, size)
	return s.repo.PageType(ctx, filter, page, size,
		bson.D{{Key: "createdAt", Value: 1}})
}

// TypeCreate 新建字典类型。
func (s *DictService) TypeCreate(ctx context.Context, in *DictTypeInput) (*model.DictType, error) {
	if n, err := s.repo.CountType(ctx, bson.M{"code": in.Code}); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, errs.BadRequest("字典编码已存在")
	}
	t := &model.DictType{Name: in.Name, Code: in.Code, Status: in.Status, Remark: in.Remark}
	t.PrepareCreate()
	if err := s.repo.InsertType(ctx, t); err != nil {
		return nil, err
	}
	s.invalidateDict(ctx)
	return t, nil
}

// TypeUpdate 更新字典类型。
func (s *DictService) TypeUpdate(ctx context.Context, id string, in *DictTypeInput) error {
	old, err := s.repo.FindType(ctx, bson.M{"_id": id})
	if err != nil {
		return errs.NotFound("字典类型不存在")
	}
	if in.Code != old.Code {
		if n, err := s.repo.CountType(ctx, bson.M{"code": in.Code, "_id": bson.M{"$ne": id}}); err != nil {
			return err
		} else if n > 0 {
			return errs.BadRequest("字典编码已存在")
		}
	}
	if err := s.repo.UpdateType(ctx, id,
		bson.M{"name": in.Name, "code": in.Code, "status": in.Status,
			"remark": in.Remark, "updatedAt": time.Now()}); err != nil {
		return err
	}
	// 编码变化时同步字典项
	if in.Code != old.Code {
		if err := s.repo.RenameItemsTypeCode(ctx, old.Code, in.Code); err != nil {
			return err
		}
	}
	s.invalidateDict(ctx)
	return nil
}

// TypeDelete 删除字典类型及其全部字典项。
func (s *DictService) TypeDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的字典类型")
	}
	types, err := s.repo.FindTypesByIDs(ctx, ids)
	if err != nil {
		return err
	}
	codes := make([]string, 0, len(types))
	for _, t := range types {
		codes = append(codes, t.Code)
	}
	if _, err := s.repo.DeleteItemsByTypeCodes(ctx, codes); err != nil {
		return err
	}
	if _, err := s.repo.DeleteTypes(ctx, ids); err != nil {
		return err
	}
	s.invalidateDict(ctx)
	return nil
}

// ItemList 字典项分页。
func (s *DictService) ItemList(ctx context.Context, typeCode string, page, size int) ([]*model.DictItem, int64, error) {
	filter := bson.M{}
	if typeCode != "" {
		filter["typeCode"] = typeCode
	}
	page, size = normalizePage(page, size)
	return s.repo.PageItem(ctx, filter, page, size,
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
}

// ItemCreate 新建字典项。
func (s *DictService) ItemCreate(ctx context.Context, in *DictItemInput) (*model.DictItem, error) {
	if n, err := s.repo.CountItem(ctx,
		bson.M{"typeCode": in.TypeCode, "value": in.Value}); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, errs.BadRequest("该字典下已存在相同键值")
	}
	i := &model.DictItem{
		TypeCode: in.TypeCode, Label: in.Label, Value: in.Value,
		TagType: in.TagType, Sort: in.Sort, Status: in.Status, Remark: in.Remark,
	}
	i.PrepareCreate()
	if err := s.repo.InsertItem(ctx, i); err != nil {
		return nil, err
	}
	s.invalidateDict(ctx)
	return i, nil
}

// ItemUpdate 更新字典项。
func (s *DictService) ItemUpdate(ctx context.Context, id string, in *DictItemInput) error {
	if n, err := s.repo.CountItem(ctx,
		bson.M{"typeCode": in.TypeCode, "value": in.Value, "_id": bson.M{"$ne": id}}); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("该字典下已存在相同键值")
	}
	if err := s.repo.UpdateItem(ctx, id,
		bson.M{"label": in.Label, "value": in.Value, "tagType": in.TagType,
			"sort": in.Sort, "status": in.Status, "remark": in.Remark, "updatedAt": time.Now()}); err != nil {
		return err
	}
	s.invalidateDict(ctx)
	return nil
}

// ItemDelete 批量删除字典项。
func (s *DictService) ItemDelete(ctx context.Context, ids []string) error {
	if _, err := s.repo.DeleteItems(ctx, ids); err != nil {
		return err
	}
	s.invalidateDict(ctx)
	return nil
}

// GetByCode 按字典编码取启用的字典项（供下拉框使用）。
// PUBLIC 策略：字典数据全用户共享（如国家/货币），Key: serveradmin:v1:dict:{code}。
func (s *DictService) GetByCode(ctx context.Context, code string) ([]*model.DictItem, error) {
	return cache.GetJSON(s.cache, ctx, cache.PublicKey("dict", code), s.cache.TTL.Public,
		func(ctx context.Context) ([]*model.DictItem, error) {
			return s.repo.FindEnabledItemsByType(ctx, code)
		})
}

// invalidateDict 写操作成功后失效全部字典缓存变体。
func (s *DictService) invalidateDict(ctx context.Context) {
	s.cache.Invalidate(ctx, cache.BizPrefix("dict"))
}
