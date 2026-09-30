package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// JobRepository 定时任务与执行日志数据访问。
type JobRepository struct {
	jobs collection[model.Job]
	logs collection[model.JobLog]
}

// NewJobRepository 创建定时任务仓库。
func NewJobRepository(db *mongo.Database) *JobRepository {
	return &JobRepository{
		jobs: collectionOf[model.Job](db, model.ColJob),
		logs: collectionOf[model.JobLog](db, model.ColJobLog),
	}
}

// FindEnabled 取全部启用的任务（调度器启动用）。
func (r *JobRepository) FindEnabled(ctx context.Context) ([]*model.Job, error) {
	return r.jobs.FindAll(ctx, bson.M{"status": model.StatusEnabled}, nil)
}

// PageJob 任务分页。
func (r *JobRepository) PageJob(ctx context.Context, filter bson.M, page, size int) ([]*model.Job, int64, error) {
	return r.jobs.Page(ctx, filter, page, size, nil)
}

// FindJob 按过滤条件取任务。
func (r *JobRepository) FindJob(ctx context.Context, filter bson.M) (*model.Job, error) {
	return r.jobs.FindOne(ctx, filter)
}

// CountJob 统计任务。
func (r *JobRepository) CountJob(ctx context.Context, filter bson.M) (int64, error) {
	return r.jobs.Count(ctx, filter)
}

// InsertJob 新建任务。
func (r *JobRepository) InsertJob(ctx context.Context, j *model.Job) error {
	return r.jobs.Insert(ctx, j)
}

// UpdateJob 更新任务。
func (r *JobRepository) UpdateJob(ctx context.Context, id string, set bson.M) error {
	return r.jobs.UpdateSet(ctx, bson.M{"_id": id}, set)
}

// UpdateJobRaw 按过滤条件执行任意任务更新（$inc/$set 组合）。
func (r *JobRepository) UpdateJobRaw(ctx context.Context, filter, update bson.M) error {
	return r.jobs.Update(ctx, filter, update)
}

// DeleteJobs 批量删除任务。
func (r *JobRepository) DeleteJobs(ctx context.Context, ids []string) (int64, error) {
	return r.jobs.DeleteByIDs(ctx, ids)
}

// ---------- 执行日志 ----------

// InsertJobLog 保存执行日志。
func (r *JobRepository) InsertJobLog(ctx context.Context, log *model.JobLog) error {
	return r.logs.Insert(ctx, log)
}

// PageJobLog 执行日志分页。
func (r *JobRepository) PageJobLog(ctx context.Context, filter bson.M, page, size int) ([]*model.JobLog, int64, error) {
	return r.logs.Page(ctx, filter, page, size, nil)
}
