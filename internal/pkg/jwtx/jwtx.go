// Package jwtx 封装 JWT 令牌的签发与解析（访问令牌 + 刷新令牌）。
package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// 令牌类型。
const (
	TypeAccess  = "access"
	TypeRefresh = "refresh"
)

// Claims 自定义声明。
type Claims struct {
	UserID   string   `json:"uid"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Roles    []string `json:"roles"`
	Type     string   `json:"typ"`
	jwt.RegisteredClaims
}

// TokenUser 签发令牌所需的用户信息。
type TokenUser struct {
	ID       string
	Username string
	Nickname string
	Roles    []string
}

// Manager JWT 管理器。
type Manager struct {
	secret        []byte
	issuer        string
	accessExpire  time.Duration
	refreshExpire time.Duration
}

// NewManager 创建管理器。
func NewManager(secret, issuer string, accessExpire, refreshExpire time.Duration) *Manager {
	return &Manager{
		secret:        []byte(secret),
		issuer:        issuer,
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}
}

// AccessExpire 访问令牌有效期。
func (m *Manager) AccessExpire() time.Duration { return m.accessExpire }

// RefreshExpire 刷新令牌有效期。
func (m *Manager) RefreshExpire() time.Duration { return m.refreshExpire }

// Generate 为用户签发一对令牌，返回 (access, refresh, 访问令牌过期时间)。
func (m *Manager) Generate(u *TokenUser) (string, string, time.Time, error) {
	now := time.Now()
	accessExp := now.Add(m.accessExpire)
	refreshExp := now.Add(m.refreshExpire)

	access, err := m.sign(&Claims{
		UserID: u.ID, Username: u.Username, Nickname: u.Nickname,
		Roles: u.Roles, Type: TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        bson.NewObjectID().Hex(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	})
	if err != nil {
		return "", "", time.Time{}, err
	}

	refresh, err := m.sign(&Claims{
		UserID: u.ID, Username: u.Username, Nickname: u.Nickname,
		Roles: u.Roles, Type: TypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        bson.NewObjectID().Hex(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	})
	if err != nil {
		return "", "", time.Time{}, err
	}
	return access, refresh, accessExp, nil
}

func (m *Manager) sign(claims *Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 解析并校验令牌，typ 指定期望的令牌类型。
func (m *Manager) Parse(tokenStr, typ string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
	)
	if err != nil {
		return nil, errors.New("令牌无效或已过期")
	}
	if claims.Type != typ {
		return nil, errors.New("令牌类型不匹配")
	}
	return claims, nil
}
