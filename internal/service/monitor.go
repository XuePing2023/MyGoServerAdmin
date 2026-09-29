package service

import (
	"context"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// MonitorService 系统监控。
type MonitorService struct {
	startAt time.Time
}

// NewMonitorService 创建监控服务。
func NewMonitorService() *MonitorService {
	return &MonitorService{startAt: time.Now()}
}

// ServerInfo 服务器监控信息。
type ServerInfo struct {
	// 主机
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	Platform   string `json:"platform"`
	KernelArch string `json:"kernelArch"`
	HostUptime uint64 `json:"hostUptime"`

	// CPU
	CPUCores   int     `json:"cpuCores"`
	CPUPercent float64 `json:"cpuPercent"`

	// 内存
	MemTotal   uint64  `json:"memTotal"`
	MemUsed    uint64  `json:"memUsed"`
	MemPercent float64 `json:"memPercent"`

	// 磁盘（应用所在盘）
	DiskPath    string  `json:"diskPath"`
	DiskTotal   uint64  `json:"diskTotal"`
	DiskFree    uint64  `json:"diskFree"`
	DiskPercent float64 `json:"diskPercent"`

	// 运行时
	GoVersion    string  `json:"goVersion"`
	GinVersion   string  `json:"ginVersion"`
	NumGoroutine int     `json:"numGoroutine"`
	HeapAllocMB  float64 `json:"heapAllocMB"`
	AppUptimeS   float64 `json:"appUptimeS"`
	Now          string  `json:"now"`
}

// Info 采集服务器信息。cpuPercent 若 >0 则直接使用（由 API 层采样传入）。
func (s *MonitorService) Info(ctx context.Context, cpuPercent float64) (*ServerInfo, error) {
	info := &ServerInfo{
		CPUCores:     runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		GinVersion:   gin.Version,
		NumGoroutine: runtime.NumGoroutine(),
		AppUptimeS:   time.Since(s.startAt).Seconds(),
		Now:          time.Now().Format(time.DateTime),
	}

	if hi, err := host.Info(); err == nil {
		info.Hostname = hi.Hostname
		info.OS = hi.OS
		info.Platform = hi.Platform + " " + hi.PlatformVersion
		info.KernelArch = hi.KernelArch
		info.HostUptime = hi.Uptime
	}

	if cp, err := cpu.Percent(0, false); err == nil && len(cp) > 0 {
		info.CPUPercent = round1(cp[0])
	} else {
		info.CPUPercent = cpuPercent
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		info.MemTotal = vm.Total
		info.MemUsed = vm.Used
		info.MemPercent = round1(vm.UsedPercent)
	}

	if du, err := disk.Usage("."); err == nil {
		info.DiskPath = du.Path
		info.DiskTotal = du.Total
		info.DiskFree = du.Free
		info.DiskPercent = round1(du.UsedPercent)
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	info.HeapAllocMB = float64(ms.HeapAlloc) / 1024 / 1024
	info.HeapAllocMB = round1(info.HeapAllocMB)

	return info, nil
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
