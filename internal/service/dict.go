package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
)

// DictService 字典管理。
type DictService struct {
	db *mongo.Database
}

// NewDictService 创建字典服务。
func NewDictService(db *mongo.Database) *DictService {
	return &DictService{db: db}
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
	return pageFind[model.DictType](ctx, s.db.Collection(model.ColDictType), filter, page, size,
		bson.D{{Key: "createdAt", Value: 1}})
}

// TypeCreate 新建字典类型。
func (s *DictService) TypeCreate(ctx context.Context, in *DictTypeInput) (*model.DictType, error) {
	if n, err := count(ctx, s.db.Collection(model.ColDictType), bson.M{"code": in.Code}); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, errs.BadRequest("字典编码已存在")
	}
	t := &model.DictType{Name: in.Name, Code: in.Code, Status: in.Status, Remark: in.Remark}
	t.PrepareCreate()
	_, err := s.db.Collection(model.ColDictType).InsertOne(ctx, t)
	return t, err
}

// TypeUpdate 更新字典类型。
func (s *DictService) TypeUpdate(ctx context.Context, id string, in *DictTypeInput) error {
	var old model.DictType
	if err := findOne(ctx, s.db.Collection(model.ColDictType), bson.M{"_id": id}, &old); err != nil {
		return errs.NotFound("字典类型不存在")
	}
	if in.Code != old.Code {
		if n, err := count(ctx, s.db.Collection(model.ColDictType), bson.M{"code": in.Code, "_id": bson.M{"$ne": id}}); err != nil {
			return err
		} else if n > 0 {
			return errs.BadRequest("字典编码已存在")
		}
	}
	_, err := s.db.Collection(model.ColDictType).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"name": in.Name, "code": in.Code, "status": in.Status,
			"remark": in.Remark, "updatedAt": time.Now()}})
	if err != nil {
		return err
	}
	// 编码变化时同步字典项
	if in.Code != old.Code {
		_, err = s.db.Collection(model.ColDictItem).UpdateMany(ctx,
			bson.M{"typeCode": old.Code},
			bson.M{"$set": bson.M{"typeCode": in.Code}})
	}
	return err
}

// TypeDelete 删除字典类型及其全部字典项。
func (s *DictService) TypeDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的字典类型")
	}
	types, err := s.db.Collection(model.ColDictType).Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return err
	}
	list := make([]*model.DictType, 0)
	if err := types.All(ctx, &list); err != nil {
		types.Close(ctx)
		return err
	}
	types.Close(ctx)
	codes := make([]string, 0, len(list))
	for _, t := range list {
		codes = append(codes, t.Code)
	}
	if _, err := s.db.Collection(model.ColDictItem).DeleteMany(ctx, bson.M{"typeCode": bson.M{"$in": codes}}); err != nil {
		return err
	}
	_, err = s.db.Collection(model.ColDictType).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}

// ItemList 字典项分页。
func (s *DictService) ItemList(ctx context.Context, typeCode string, page, size int) ([]*model.DictItem, int64, error) {
	filter := bson.M{}
	if typeCode != "" {
		filter["typeCode"] = typeCode
	}
	page, size = normalizePage(page, size)
	return pageFind[model.DictItem](ctx, s.db.Collection(model.ColDictItem), filter, page, size,
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
}

// ItemCreate 新建字典项。
func (s *DictService) ItemCreate(ctx context.Context, in *DictItemInput) (*model.DictItem, error) {
	if n, err := count(ctx, s.db.Collection(model.ColDictItem),
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
	_, err := s.db.Collection(model.ColDictItem).InsertOne(ctx, i)
	return i, err
}

// ItemUpdate 更新字典项。
func (s *DictService) ItemUpdate(ctx context.Context, id string, in *DictItemInput) error {
	if n, err := count(ctx, s.db.Collection(model.ColDictItem),
		bson.M{"typeCode": in.TypeCode, "value": in.Value, "_id": bson.M{"$ne": id}}); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("该字典下已存在相同键值")
	}
	_, err := s.db.Collection(model.ColDictItem).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"label": in.Label, "value": in.Value, "tagType": in.TagType,
			"sort": in.Sort, "status": in.Status, "remark": in.Remark, "updatedAt": time.Now()}})
	return err
}

// ItemDelete 批量删除字典项。
func (s *DictService) ItemDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的字典项")
	}
	_, err := s.db.Collection(model.ColDictItem).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}

// GetByCode 按字典编码取启用的字典项（供下拉框使用）。
func (s *DictService) GetByCode(ctx context.Context, code string) ([]*model.DictItem, error) {
	opts := options.Find().SetSort(bson.D{{Key: "sort", Value: 1}})
	cursor, err := s.db.Collection(model.ColDictItem).Find(ctx,
		bson.M{"typeCode": code, "status": model.StatusEnabled}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]*model.DictItem, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}
