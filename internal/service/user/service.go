package user

import (
	"context"
	"strings"
	"time"

	"github.com/haowen-xu/agent-coder/internal/auth"
	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/utils"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// AuthUser 表示数据结构定义。
type AuthUser struct {
	ID       uint   // ID 字段说明。
	Username string // Username 字段说明。
	IsAdmin  bool   // IsAdmin 字段说明。
	Enabled  bool   // Enabled 字段说明。
}

// Service 表示数据结构定义。
type Service struct {
	cfg *appcfg.Config // cfg 字段说明。
	db  *db.Client     // db 字段说明。
}

// New 执行相关逻辑。
func New(cfg *appcfg.Config, dbClient *db.Client) *Service {
	return &Service{
		cfg: cfg,
		db:  dbClient,
	}
}

// Login 是 *Service 的方法实现。
func (s *Service) Login(ctx context.Context, username string, password string) (string, time.Time, *AuthUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", time.Time{}, nil, xerr.Config.New("username/password are required")
	}

	user, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return "", time.Time{}, nil, err
	}
	if user == nil || !user.Enabled {
		return "", time.Time{}, nil, xerr.Config.New("invalid username or password")
	}
	if !auth.VerifyPassword(password, user.PasswordHash) {
		return "", time.Time{}, nil, xerr.Config.New("invalid username or password")
	}

	token, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, nil, xerr.Infra.Wrap(err, "new session token")
	}
	now := utils.NowUTC()
	expiredAt := now.Add(s.cfg.Auth.SessionTTLDuration())
	sess := &db.UserSession{
		UserID:    user.ID,
		Token:     token,
		ExpiredAt: expiredAt,
	}
	if err := s.db.CreateSession(ctx, sess); err != nil {
		return "", time.Time{}, nil, err
	}
	user.LastLoginAt = &now
	if err := s.db.SaveUser(ctx, user); err != nil {
		return "", time.Time{}, nil, err
	}
	return token, expiredAt, &AuthUser{
		ID:       user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Enabled:  user.Enabled,
	}, nil
}

// AuthByToken 是 *Service 的方法实现。
func (s *Service) AuthByToken(ctx context.Context, token string) (*AuthUser, error) {
	_, user, err := s.db.GetSessionWithUser(ctx, token)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.Enabled {
		return nil, nil
	}
	return &AuthUser{
		ID:       user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Enabled:  user.Enabled,
	}, nil
}

// ListUsers 是 *Service 的方法实现。
func (s *Service) ListUsers(ctx context.Context) ([]db.User, error) {
	return s.db.ListUsers(ctx)
}

// CreateUser 是 *Service 的方法实现。
func (s *Service) CreateUser(ctx context.Context, username string, password string, isAdmin bool, enabled bool) (*db.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, xerr.Config.New("username/password are required")
	}
	existed, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existed != nil {
		return nil, xerr.Config.New("username already exists")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "hash password")
	}
	row := &db.User{
		Username:     username,
		PasswordHash: hash,
		IsAdmin:      isAdmin,
		Enabled:      enabled,
	}
	if err := s.db.CreateUser(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateUser 是 *Service 的方法实现。
func (s *Service) UpdateUser(ctx context.Context, userID uint, password *string, isAdmin *bool, enabled *bool) (*db.User, error) {
	row, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, xerr.Config.New("user not found")
	}
	if password != nil {
		hash, err := auth.HashPassword(*password)
		if err != nil {
			return nil, xerr.Infra.Wrap(err, "hash password")
		}
		row.PasswordHash = hash
	}
	if isAdmin != nil {
		row.IsAdmin = *isAdmin
	}
	if enabled != nil {
		row.Enabled = *enabled
	}
	if err := s.db.SaveUser(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}
