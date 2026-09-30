package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// DictRepository 字典类型与字典项数据访问。
type DictRepository struct {
	types collection[model.DictType]
	items collection[model.DictItem]
}

// NewDictRepository 创建字典仓库。
func NewDictRepository(db *mongo.Database) *DictRepository {
	return &DictRepository{
		types: collectionOf[model.DictType](db, model.ColDictType),
		items: collectionOf[model.DictItem](db, model.ColDictItem),
	}
}

// ---------- 字典类型 ----------

// CountType 统计字典类型。
func (r *DictRepository) CountType(ctx context.Context, filter bson.M) (int64, error) {
	return r.types.Count(ctx, filter)
}

// FindType 按过滤条件取字典类型。
func (r *DictRepository) FindType(ctx context.Context, filter bson.M) (*model.DictType, error) {
	return r.types.FindOne(ctx, filter)
}

// FindTypesByIDs 按主键批量取字典类型。
func (r *DictRepository) FindTypesByIDs(ctx context.Context, ids []string) ([]*model.DictType, error) {
	return r.types.FindByIDs(ctx, ids)
}

// PageType 字典类型分页。
func (r *DictRepository) PageType(ctx context.Context, filter bson.M, page, size int, sort bson.D) ([]*model.DictType, int64, error) {
	return r.types.Page(ctx, filter, page, size, sort)
}

// InsertType 新建字典类型。
func (r *DictRepository) InsertType(ctx context.Context, t *model.DictType) error {
	return r.types.Insert(ctx, t)
}

// UpdateType 更新字典类型。
func (r *DictRepository) UpdateType(ctx context.Context, id string, set bson.M) error {
	return r.types.UpdateSet(ctx, bson.M{"_id": id}, set)
}

// DeleteTypes 批量删除字典类型。
func (r *DictRepository) DeleteTypes(ctx context.Context, ids []string) (int64, error) {
	return r.types.DeleteByIDs(ctx, ids)
}

// ---------- 字典项 ----------

// CountItem 统计字典项。
func (r *DictRepository) CountItem(ctx context.Context, filter bson.M) (int64, error) {
	return r.items.Count(ctx, filter)
}

// PageItem 字典项分页。
func (r *DictRepository) PageItem(ctx context.Context, filter bson.M, page, size int, sort bson.D) ([]*model.DictItem, int64, error) {
	return r.items.Page(ctx, filter, page, size, sort)
}

// FindEnabledItemsByType 取某类型下启用的字典项（按 sort 排序，下拉框用）。
func (r *DictRepository) FindEnabledItemsByType(ctx context.Context, typeCode string) ([]*model.DictItem, error) {
	return r.items.FindAll(ctx,
		bson.M{"typeCode": typeCode, "status": model.StatusEnabled},
		bson.D{{Key: "sort", Value: 1}})
}

// InsertItem 新建字典项。
func (r *DictRepository) InsertItem(ctx context.Context, i *model.DictItem) error {
	return r.items.Insert(ctx, i)
}

// UpdateItem 更新字典项。
func (r *DictRepository) UpdateItem(ctx context.Context, id string, set bson.M) error {
	return r.items.UpdateSet(ctx, bson.M{"_id": id}, set)
}

// DeleteItems 批量删除字典项。
func (r *DictRepository) DeleteItems(ctx context.Context, ids []string) (int64, error) {
	return r.items.DeleteByIDs(ctx, ids)
}

// DeleteItemsByTypeCodes 删除这些类型下的全部字典项。
func (r *DictRepository) DeleteItemsByTypeCodes(ctx context.Context, codes []string) (int64, error) {
	if len(codes) == 0 {
		return 0, nil
	}
	return r.items.Delete(ctx, bson.M{"typeCode": bson.M{"$in": codes}})
}

// RenameItemsTypeCode 字典类型编码变化时同步字典项。
func (r *DictRepository) RenameItemsTypeCode(ctx context.Context, oldCode, newCode string) error {
	_, err := r.items.coll.UpdateMany(ctx,
		bson.M{"typeCode": oldCode},
		bson.M{"$set": bson.M{"typeCode": newCode}})
	return err
}
