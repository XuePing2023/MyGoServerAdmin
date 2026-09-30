package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/logger"
)

// HandlerFunc 定时任务处理器：ctx / 数据库 / JSON 参数。
type HandlerFunc func(ctx context.Context, db *mongo.Database, params string) (string, error)

// handlerRegistry 内置任务处理器注册表。
var handlerRegistry = map[string]HandlerFunc{
	"cleanup_expired_tokens": handleCleanupExpiredTokens,
	"cleanup_operation_logs": handleCleanupOperationLogs,
	"cleanup_login_logs":     handleCleanupLoginLogs,
	"demo_heartbeat":         handleDemoHeartbeat,
}

// HandlerCatalog 供前端下拉选择：名称 -> 描述。
func HandlerCatalog() map[string]string {
	return map[string]string{
		"cleanup_expired_tokens": "清理过期令牌黑名单",
		"cleanup_operation_logs": "清理历史操作日志（params: {\"days\":90}）",
		"cleanup_login_logs":     "清理历史登录日志（params: {\"days\":180}）",
		"demo_heartbeat":         "演示：心跳输出（无参数）",
	}
}

func handleCleanupExpiredTokens(ctx context.Context, db *mongo.Database, params string) (string, error) {
	res, err := db.Collection(model.ColTokenBlacklist).DeleteMany(ctx,
		bson.M{"expireAt": bson.M{"$lte": time.Now()}})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已清理过期令牌 %d 条", res.DeletedCount), nil
}

func handleCleanupOperationLogs(ctx context.Context, db *mongo.Database, params string) (string, error) {
	days := parseDaysParam(params, 90)
	cut := time.Now().AddDate(0, 0, -days)
	res, err := db.Collection(model.ColOperationLog).DeleteMany(ctx,
		bson.M{"createdAt": bson.M{"$lt": cut}})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已清理 %d 天前的操作日志 %d 条", days, res.DeletedCount), nil
}

func handleCleanupLoginLogs(ctx context.Context, db *mongo.Database, params string) (string, error) {
	days := parseDaysParam(params, 180)
	cut := time.Now().AddDate(0, 0, -days)
	res, err := db.Collection(model.ColLoginLog).DeleteMany(ctx,
		bson.M{"loginAt": bson.M{"$lt": cut}})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已清理 %d 天前的登录日志 %d 条", days, res.DeletedCount), nil
}

func handleDemoHeartbeat(ctx context.Context, db *mongo.Database, params string) (string, error) {
	msg := fmt.Sprintf("heartbeat ok at %s", time.Now().Format(time.DateTime))
	logger.L.Info("定时任务心跳: ", msg)
	return msg, nil
}

func parseDaysParam(params string, def int) int {
	if params == "" {
		return def
	}
	var m struct {
		Days int `json:"days"`
	}
	if err := json.Unmarshal([]byte(params), &m); err == nil && m.Days > 0 {
		return m.Days
	}
	return def
}

// JobService 定时任务管理（基于 robfig/cron）。
type JobService struct {
	db      *mongo.Database
	cron    *cron.Cron
	mu      sync.Mutex
	entries map[string]cron.EntryID // jobID -> cron entry
}

// NewJobService 创建定时任务服务。
func NewJobService(db *mongo.Database) *JobService {
	return &JobService{
		db:      db,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
	}
}

// Start 从数据库加载启用的任务并启动调度器。
func (s *JobService) Start(ctx context.Context) error {
	cursor, err := s.db.Collection(model.ColJob).Find(ctx, bson.M{"status": model.StatusEnabled})
	if err != nil {
		return err
	}
	var jobs []*model.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		cursor.Close(ctx)
		return err
	}
	cursor.Close(ctx)

	for _, j := range jobs {
		if err := s.schedule(j); err != nil {
			logger.L.Errorf("加载定时任务 %s 失败: %v", j.Name, err)
		}
	}
	s.cron.Start()
	logger.L.Infof("定时任务调度器已启动，运行中任务 %d 个", len(s.cron.Entries()))
	return nil
}

// Stop 停止调度器（等待执行中的任务结束）。
func (s *JobService) Stop() {
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
	}
}

// JobInput 任务创建/更新。
type JobInput struct {
	Name    string `json:"name" binding:"required,max=64"`
	Spec    string `json:"spec" binding:"required"`
	Handler string `json:"handler" binding:"required"`
	Params  string `json:"params"`
	Status  int    `json:"status" binding:"required,oneof=1 2"`
	Remark  string `json:"remark"`
}

// validate 校验 cron 表达式与处理器。
func (in *JobInput) validate() error {
	if _, err := cron.ParseStandard(in.Spec); err != nil {
		return errs.BadRequest("cron 表达式无效（示例: 0 2 * * *）: %v", err)
	}
	if _, ok := handlerRegistry[in.Handler]; !ok {
		return errs.BadRequest("任务处理器 %s 未注册", in.Handler)
	}
	return nil
}

// List 任务分页。
func (s *JobService) List(ctx context.Context, name string, status, page, size int) ([]*model.Job, int64, error) {
	filter := statusFilter(status)
	if name != "" {
		filter["name"] = likeFilter(name)
	}
	page, size = normalizePage(page, size)
	return pageFind[model.Job](ctx, s.db.Collection(model.ColJob), filter, page, size, nil)
}

// Create 创建任务。
func (s *JobService) Create(ctx context.Context, in *JobInput) (*model.Job, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	if n, err := count(ctx, s.db.Collection(model.ColJob), bson.M{"name": in.Name}); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, errs.BadRequest("任务名称已存在")
	}
	j := &model.Job{
		Name: in.Name, Spec: in.Spec, Handler: in.Handler,
		Params: in.Params, Status: in.Status, Remark: in.Remark,
	}
	j.PrepareCreate()
	if _, err := s.db.Collection(model.ColJob).InsertOne(ctx, j); err != nil {
		return nil, err
	}
	if j.Status == model.StatusEnabled {
		if err := s.schedule(j); err != nil {
			return nil, err
		}
	}
	return j, nil
}

// Update 更新任务。
func (s *JobService) Update(ctx context.Context, id string, in *JobInput) error {
	if _, err := s.get(ctx, id); err != nil {
		return err
	}
	if n, err := count(ctx, s.db.Collection(model.ColJob), bson.M{"name": in.Name, "_id": bson.M{"$ne": id}}); err != nil {
		return err
	} else if n > 0 {
		return errs.BadRequest("任务名称已存在")
	}
	if err := in.validate(); err != nil {
		return err
	}
	_, err := s.db.Collection(model.ColJob).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"name": in.Name, "spec": in.Spec, "handler": in.Handler,
			"params": in.Params, "status": in.Status, "remark": in.Remark, "updatedAt": time.Now()}})
	if err != nil {
		return err
	}
	s.unschedule(id)
	if in.Status == model.StatusEnabled {
		updated, err := s.get(ctx, id)
		if err != nil {
			return err
		}
		return s.schedule(updated)
	}
	return nil
}

// Delete 批量删除任务。
func (s *JobService) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的任务")
	}
	for _, id := range ids {
		s.unschedule(id)
	}
	_, err := s.db.Collection(model.ColJob).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}

// SetStatus 启用/停用任务。
func (s *JobService) SetStatus(ctx context.Context, id string, status int) error {
	if _, err := s.get(ctx, id); err != nil {
		return err
	}
	if status != model.StatusEnabled && status != model.StatusDisabled {
		return errs.BadRequest("状态值无效")
	}
	_, err := s.db.Collection(model.ColJob).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now()}})
	if err != nil {
		return err
	}
	s.unschedule(id)
	if status == model.StatusEnabled {
		updated, err := s.get(ctx, id)
		if err != nil {
			return err
		}
		return s.schedule(updated)
	}
	return nil
}

// RunOnce 立即执行一次。
func (s *JobService) RunOnce(ctx context.Context, id string) error {
	j, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if _, ok := handlerRegistry[j.Handler]; !ok {
		return errs.BadRequest("任务处理器 %s 未注册", j.Handler)
	}
	go s.execute(j, true)
	return nil
}

// JobLogs 任务执行日志分页。
func (s *JobService) JobLogs(ctx context.Context, jobName string, page, size int) ([]*model.JobLog, int64, error) {
	filter := bson.M{}
	if jobName != "" {
		filter["jobName"] = jobName
	}
	page, size = normalizePage(page, size)
	return pageFind[model.JobLog](ctx, s.db.Collection(model.ColJobLog), filter, page, size, nil)
}

// schedule 注册任务到调度器并更新下次执行时间。
func (s *JobService) schedule(j *model.Job) error {
	if _, ok := handlerRegistry[j.Handler]; !ok {
		return errs.BadRequest("任务处理器 %s 未注册", j.Handler)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, running := s.entries[j.ID]; running {
		return nil
	}
	jobID := j.ID
	entryID, err := s.cron.AddFunc(j.Spec, func() {
		s.runByName(jobID, false)
	})
	if err != nil {
		return err
	}
	s.entries[j.ID] = entryID
	entry := s.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		next := entry.Next
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.db.Collection(model.ColJob).UpdateOne(bg, bson.M{"_id": j.ID},
			bson.M{"$set": bson.M{"nextRun": next}})
	}
	return nil
}

// unschedule 从调度器移除任务。
func (s *JobService) unschedule(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.entries[jobID]; ok {
		s.cron.Remove(id)
		delete(s.entries, jobID)
	}
}

func (s *JobService) runByName(jobID string, manual bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	j, err := s.get(ctx, jobID)
	if err != nil || (j.Status != model.StatusEnabled && !manual) {
		return
	}
	s.execute(j, manual)
}

// execute 执行任务并记录日志。
func (s *JobService) execute(j *model.Job, manual bool) {
	defer func() {
		if r := recover(); r != nil {
			logger.L.Errorf("定时任务 %s panic: %v", j.Name, r)
		}
	}()
	handler := handlerRegistry[j.Handler]
	start := time.Now()
	output, err := handler(context.Background(), s.db, j.Params)

	log := &model.JobLog{
		JobName: j.Name, Handler: j.Handler, RunAt: start,
		CostMs: time.Since(start).Milliseconds(), Manual: manual,
	}
	if err != nil {
		log.Error = err.Error()
	} else {
		log.Success = true
		log.Output = output
	}
	log.PrepareCreate()

	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.db.Collection(model.ColJobLog).InsertOne(bg, log)

	inc := bson.M{"runCount": 1, "lastRun": start}
	if err != nil {
		inc["failCount"] = 1
	}
	_, _ = s.db.Collection(model.ColJob).UpdateOne(bg, bson.M{"_id": j.ID},
		bson.M{"$inc": inc, "$set": bson.M{"updatedAt": time.Now()}})

	if err != nil {
		logger.L.Errorf("定时任务 %s 执行失败: %v", j.Name, err)
	} else {
		logger.L.Infof("定时任务 %s 执行完成: %s", j.Name, output)
	}
}

func (s *JobService) get(ctx context.Context, id string) (*model.Job, error) {
	var j model.Job
	if err := findOne(ctx, s.db.Collection(model.ColJob), bson.M{"_id": id}, &j); err != nil {
		return nil, errs.NotFound("任务不存在")
	}
	return &j, nil
}
