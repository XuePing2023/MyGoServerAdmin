package service

import (
	"sync"
	"time"
)

// OnlineUser 在线用户（基于内存，适用于单实例部署；
// 集群部署可将其替换为 Redis 实现）。
type OnlineUser struct {
	JTI        string    `json:"jti"`
	UserID     string    `json:"userId"`
	Username   string    `json:"username"`
	Nickname   string    `json:"nickname"`
	IP         string    `json:"ip"`
	LoginAt    time.Time `json:"loginAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// OnlineService 在线用户登记表。
type OnlineService struct {
	mu      sync.RWMutex
	m       map[string]*OnlineUser // jti -> user
	maxIdle time.Duration
}

// NewOnlineService 创建服务并启动过期清理。
func NewOnlineService(maxIdle time.Duration) *OnlineService {
	if maxIdle <= 0 {
		maxIdle = 2 * time.Hour
	}
	s := &OnlineService{m: make(map[string]*OnlineUser), maxIdle: maxIdle}
	go s.janitor()
	return s
}

// Touch 登录或每次请求时刷新在线状态。
func (s *OnlineService) Touch(jti, userID, username, nickname, ip string) {
	if jti == "" || userID == "" {
		return
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.m[jti]; ok {
		u.LastSeenAt = now
		u.IP = ip
		return
	}
	s.m[jti] = &OnlineUser{
		JTI: jti, UserID: userID, Username: username, Nickname: nickname,
		IP: ip, LoginAt: now, LastSeenAt: now,
	}
}

// List 返回全部在线用户（按登录时间倒序）。
func (s *OnlineService) List() []*OnlineUser {
	s.mu.RLock()
	list := make([]*OnlineUser, 0, len(s.m))
	for _, u := range s.m {
		list = append(list, u)
	}
	s.mu.RUnlock()
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].LoginAt.After(list[i].LoginAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	return list
}

// Remove 移除指定会话（登出/踢下线），返回是否存在。
func (s *OnlineService) Remove(jti string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[jti]; ok {
		delete(s.m, jti)
		return true
	}
	return false
}

func (s *OnlineService) janitor() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		deadline := time.Now().Add(-s.maxIdle)
		s.mu.Lock()
		for k, u := range s.m {
			if u.LastSeenAt.Before(deadline) {
				delete(s.m, k)
			}
		}
		s.mu.Unlock()
	}
}
