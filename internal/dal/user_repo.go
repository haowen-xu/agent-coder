package db

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/auth"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// GetUserByUsername 是 *Client 的方法实现。
func (c *Client) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row User
	err := c.db.WithContext(ctx).Where("username = ?", username).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query user by username")
	}
	return &row, nil
}

// GetUserByID 是 *Client 的方法实现。
func (c *Client) GetUserByID(ctx context.Context, id uint) (*User, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row User
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query user by id")
	}
	return &row, nil
}

// ListUsers 是 *Client 的方法实现。
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var rows []User
	err := c.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list users")
	}
	return rows, nil
}

// CreateUser 是 *Client 的方法实现。
func (c *Client) CreateUser(ctx context.Context, row *User) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "create user")
	}
	return nil
}

// SaveUser 是 *Client 的方法实现。
func (c *Client) SaveUser(ctx context.Context, row *User) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Save(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "save user")
	}
	return nil
}

// CreateSession 是 *Client 的方法实现。
func (c *Client) CreateSession(ctx context.Context, row *UserSession) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "create session")
	}
	return nil
}

// GetSessionWithUser 是 *Client 的方法实现。
func (c *Client) GetSessionWithUser(ctx context.Context, token string) (*UserSession, *User, error) {
	if c == nil || c.db == nil {
		return nil, nil, xerr.Infra.New("db is not initialized")
	}
	var sess UserSession
	err := c.db.WithContext(ctx).Where("token = ?", token).First(&sess).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, xerr.Infra.Wrap(err, "query session")
	}
	if sess.ExpiredAt.Before(time.Now()) {
		return nil, nil, nil
	}

	var user User
	err = c.db.WithContext(ctx).Where("id = ?", sess.UserID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, xerr.Infra.Wrap(err, "query session user")
	}
	return &sess, &user, nil
}

// EnsureBootstrapAdmin 是 *Client 的方法实现。
func (c *Client) EnsureBootstrapAdmin(ctx context.Context, username string, password string) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	row, err := c.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if row != nil {
		return nil
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return xerr.Infra.Wrap(err, "hash bootstrap admin password")
	}
	admin := &User{
		Username:     username,
		PasswordHash: passwordHash,
		IsAdmin:      true,
		Enabled:      true,
	}
	return c.CreateUser(ctx, admin)
}
