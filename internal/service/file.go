package service

import (
	"context"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/config"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/repository"
)

var unsafeExtRe = regexp.MustCompile(`[^a-zA-Z0-9.]`)

// FileService 文件上传管理（本地磁盘存储）。
type FileService struct {
	repo      *repository.FileRepository
	uploadDir string
	maxBytes  int64
}

// NewFileService 创建文件服务。
func NewFileService(repo *repository.FileRepository, cfg *config.Config) *FileService {
	return &FileService{
		repo:      repo,
		uploadDir: cfg.App.UploadDir,
		maxBytes:  int64(cfg.App.MaxUploadMB) * 1024 * 1024,
	}
}

// Upload 保存上传文件并登记记录。
func (s *FileService) Upload(ctx context.Context, fh *multipart.FileHeader, uploader string) (*model.FileRecord, error) {
	if fh.Size > s.maxBytes {
		return nil, errs.BadRequest("文件大小超过限制（最大 %d MB）", s.maxBytes/1024/1024)
	}
	ext := unsafeExtRe.ReplaceAllString(filepath.Ext(fh.Filename), "")
	if len(ext) > 16 {
		ext = ext[:16]
	}
	subdir := time.Now().Format("200601")
	storeName := model.NewID() + ext
	relPath := subdir + "/" + storeName
	absPath := filepath.Join(s.uploadDir, subdir, storeName)

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, errs.Server("创建上传目录失败: %v", err)
	}
	src, err := fh.Open()
	if err != nil {
		return nil, errs.BadRequest("读取上传文件失败")
	}
	defer src.Close()
	dst, err := os.Create(absPath)
	if err != nil {
		return nil, errs.Server("写入文件失败: %v", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, errs.Server("保存文件失败: %v", err)
	}

	rec := &model.FileRecord{
		Name:         storeName,
		OriginalName: filepath.Base(fh.Filename),
		Size:         fh.Size,
		ContentType:  fh.Header.Get("Content-Type"),
		Path:         relPath,
		URL:          "/uploads/" + relPath,
		Uploader:     uploader,
	}
	rec.PrepareCreate()
	if err := s.repo.Insert(ctx, rec); err != nil {
		_ = os.Remove(absPath)
		return nil, err
	}
	return rec, nil
}

// List 文件分页列表。
func (s *FileService) List(ctx context.Context, name string, page, size int) ([]*model.FileRecord, int64, error) {
	filter := bson.M{}
	if name != "" {
		filter["originalName"] = likeFilter(name)
	}
	page, size = normalizePage(page, size)
	return s.repo.Page(ctx, filter, page, size, nil)
}

// Get 文件记录详情。
func (s *FileService) Get(ctx context.Context, id string) (*model.FileRecord, error) {
	rec, err := s.repo.FindOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, errs.NotFound("文件不存在")
	}
	return rec, nil
}

// Delete 删除文件（磁盘 + 记录）。
func (s *FileService) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的文件")
	}
	list, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, rec := range list {
		_ = os.Remove(filepath.Join(s.uploadDir, filepath.FromSlash(rec.Path)))
	}
	_, err = s.repo.DeleteByIDs(ctx, ids)
	return err
}

// AbsPath 记录对应磁盘绝对路径。
func (s *FileService) AbsPath(rec *model.FileRecord) string {
	return filepath.Join(s.uploadDir, filepath.FromSlash(rec.Path))
}
