package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/repository"
)

// DepartmentService 部门管理。
type DepartmentService struct {
	repo *repository.DepartmentRepository
}

// NewDepartmentService 创建部门服务。
func NewDepartmentService(repo *repository.DepartmentRepository) *DepartmentService {
	return &DepartmentService{repo: repo}
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
	list, err := s.repo.FindAll(ctx, bson.M{},
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
	if err != nil {
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
	if err := s.repo.Insert(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
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
		ids, err := s.repo.SelfAndDescendantIDs(ctx, id)
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
	return s.repo.UpdateSet(ctx, bson.M{"_id": d.ID}, update)
}

// Delete 删除部门。
func (s *DepartmentService) Delete(ctx context.Context, id string) error {
	if n, err := s.repo.Count(ctx, bson.M{"parentId": id}); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("存在子部门，请先删除子部门")
	}
	if n, err := s.repo.CountUsers(ctx, id); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("该部门下仍有 %d 个用户，请先调整", n)
	}
	n, err := s.repo.Delete(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if n == 0 {
		return errs.NotFound("部门不存在")
	}
	return nil
}

func (s *DepartmentService) get(ctx context.Context, id string) (*model.Department, error) {
	d, err := s.repo.FindOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, errs.NotFound("部门不存在")
	}
	return d, nil
}

func (s *DepartmentService) ensureExists(ctx context.Context, id string) error {
	_, err := s.get(ctx, id)
	return err
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
