// Package webui 内嵌管理后台前端（纯静态资源，构建后为单二进制）。
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var staticFS embed.FS

// FS 返回前端静态资源文件系统。
func FS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return staticFS
	}
	return sub
}
