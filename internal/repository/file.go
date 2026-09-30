package repository

import (
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// FileRepository 文件记录集合数据访问。
type FileRepository struct {
	collection[model.FileRecord]
}

// NewFileRepository 创建文件仓库。
func NewFileRepository(db *mongo.Database) *FileRepository {
	return &FileRepository{collectionOf[model.FileRecord](db, model.ColFile)}
}
