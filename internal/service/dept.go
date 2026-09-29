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

// DepartmentService 部门管理。
type DepartmentService struct {
	db *mongo.Database
}

// NewDepartmentService 创建部门服务。
func NewDepartmentService(db *mongo.Database) *DepartmentService {
	return &DepartmentService{db: db}
}

// DeptInput 创建/更新部门。
type DeptInput struct {
	ParentID string `json:"parentId"`
	Name     string `json:"name" binding:"required,max=32"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status" binding:"required,oneof=1 2"`
}

// Tree 部门树。
func (s *DepartmentService) Tree(ctx context.Context) ([]*model.Department, error) {
	opts := options.Find().SetSort(bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
	cursor, err := s.db.Collection(model.ColDepartment).Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]*model.Department, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return buildDeptTree(list, ""), nil
}

// Create 创建部门。
func (s *DepartmentService) Create(ctx context.Context, in *DeptInput) (*model.Department, error) {
	if in.ParentID != "" {
		if err := s.ensureExists(ctx, in.ParentID); err != nil {
			return nil, err
		}
	}
	d := &model.Department{
		ParentID: in.ParentID, Name: in.Name, Leader: in.Leader,
		Phone: in.Phone, Sort: in.Sort, Status: in.Status,
	}
	d.PrepareCreate()
	_, err := s.db.Collection(model.ColDepartment).InsertOne(ctx, d)
	return d, err
}

// Update 更新部门。
func (s *DepartmentService) Update(ctx context.Context, id string, in *DeptInput) error {
	d, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if in.ParentID == id {
		return errs.BadRequest("上级部门不能是自身")
	}
	if in.ParentID != "" {
		if err := s.ensureExists(ctx, in.ParentID); err != nil {
			return err
		}
		ids, err := s.selfAndDescendantIDs(ctx, id)
		if err != nil {
			return err
		}
		if contains(ids, in.ParentID) {
			return errs.BadRequest("不能把自身的子部门设为上级部门")
		}
	}
	update := bson.M{
		"parentId": in.ParentID, "name": in.Name, "leader": in.Leader,
		"phone": in.Phone, "sort": in.Sort, "status": in.Status, "updatedAt": time.Now(),
	}
	_, err = s.db.Collection(model.ColDepartment).UpdateOne(ctx, bson.M{"_id": d.ID}, bson.M{"$set": update})
	return err
}

// Delete 删除部门。
func (s *DepartmentService) Delete(ctx context.Context, id string) error {
	if n, err := count(ctx, s.db.Collection(model.ColDepartment), bson.M{"parentId": id}); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("存在子部门，请先删除子部门")
	}
	if n, err := count(ctx, s.db.Collection(model.ColUser), bson.M{"deptId": id}); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("该部门下仍有 %d 个用户，请先调整", n)
	}
	res, err := s.db.Collection(model.ColDepartment).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errs.NotFound("部门不存在")
	}
	return nil
}

func (s *DepartmentService) get(ctx context.Context, id string) (*model.Department, error) {
	var d model.Department
	if err := findOne(ctx, s.db.Collection(model.ColDepartment), bson.M{"_id": id}, &d); err != nil {
		return nil, errs.NotFound("部门不存在")
	}
	return &d, nil
}

func (s *DepartmentService) ensureExists(ctx context.Context, id string) error {
	_, err := s.get(ctx, id)
	return err
}

// selfAndDescendantIDs 返回自身及全部子孙部门 ID。
func (s *DepartmentService) selfAndDescendantIDs(ctx context.Context, root string) ([]string, error) {
	result := []string{root}
	frontier := []string{root}
	for len(frontier) > 0 {
		cursor, err := s.db.Collection(model.ColDepartment).Find(ctx,
			bson.M{"parentId": bson.M{"$in": frontier}})
		if err != nil {
			return nil, err
		}
		var children []*model.Department
		if err := cursor.All(ctx, &children); err != nil {
			cursor.Close(ctx)
			return nil, err
		}
		cursor.Close(ctx)
		frontier = frontier[:0]
		for _, c := range children {
			result = append(result, c.ID)
			frontier = append(frontier, c.ID)
		}
	}
	return result, nil
}

func buildDeptTree(list []*model.Department, parentID string) []*model.Department {
	tree := make([]*model.Department, 0)
	for _, d := range list {
		if d.ParentID == parentID {
			d.Children = buildDeptTree(list, d.ID)
			tree = append(tree, d)
		}
	}
	return tree
}
