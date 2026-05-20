package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/waf-control/internal/config"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID   int64    `json:"uid"`
	Username string   `json:"usr"`
	Roles    []string `json:"roles"`
}

type Service struct {
	repo *Repository
	cfg  config.AuthConfig
}

func NewService(repo *Repository, cfg config.AuthConfig) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) Authenticate(ctx context.Context, req LoginRequest) (*TokenPair, error) {
	user, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	// 优先使用 role_key（canonical 英文 key，如 system_admin），缺省时回退到 name。
	// requireRole 中间件按 key 判定，避免中文 display name 永远命中不到 admin 短语。
	roleNames := make([]string, 0, len(roles)*2)
	for _, r := range roles {
		if r.RoleKey != "" {
			roleNames = append(roleNames, r.RoleKey)
		}
		if r.Name != "" {
			roleNames = append(roleNames, r.Name)
		}
	}

	return s.issueTokenPair(ctx, user.ID, user.Username, roleNames)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	hash := HashToken(refreshToken)

	revoked, err := s.repo.IsTokenRevoked(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("check token: %w", err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	claims, err := s.ParseAccessToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	if err := s.repo.RevokeTokenByHash(ctx, hash); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	return s.issueTokenPair(ctx, claims.UserID, claims.Username, claims.Roles)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := HashToken(refreshToken)
	return s.repo.RevokeTokenByHash(ctx, hash)
}

func (s *Service) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GetUserByID 实现 project.UserLister 接口，返回 v3 风格的用户对象作为 any。
// 用于跨包传递时避免对 identity 内部类型的直接依赖。
func (s *Service) GetUserByID(ctx context.Context, id int64) (any, error) {
	u, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return toUserV3(u), nil
}

func (s *Service) GetUserWithRoles(ctx context.Context, userID int64) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return user, nil
}

func (s *Service) issueTokenPair(ctx context.Context, userID int64, username string, roles []string) (*TokenPair, error) {
	now := time.Now()

	accessClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "waf-control",
		},
		UserID:   userID,
		Username: username,
		Roles:    roles,
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(refreshBytes)

	tokenRecord := &Token{
		UserID:    userID,
		TokenType: "refresh",
		TokenHash: HashToken(refreshToken),
		ExpiresAt: now.Add(s.cfg.RefreshTTL),
	}
	if err := s.repo.SaveToken(ctx, tokenRecord); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.TokenTTL.Seconds()),
	}, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}
