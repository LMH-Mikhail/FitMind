package manager

import (
	"context"
	"database/sql"
	"errors"
	"fitmind/backend/pkg/model"
	"strings"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

type UserManager struct {
	db *sql.DB
}

type CreateUserParams struct {
	Email        string
	PasswordHash string
	Nickname     string
}

type userScanner interface {
	Scan(dest ...any) error
}

const userFields = `
	id::text,
	COALESCE(email::text, ''),
	COALESCE(phone, ''),
	COALESCE(password_hash, ''),
	COALESCE(nickname, ''),
	COALESCE(avatar_url, ''),
	gender,
	status,
	last_login_at,
	created_at,
	updated_at
`

func NewUserManager(db *sql.DB) *UserManager {
	return &UserManager{db: db}
}

func (manager *UserManager) CreateUser(ctx context.Context, params CreateUserParams) (*model.User, error) {
	tx, err := manager.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	row := tx.QueryRowContext(
		ctx,
		`INSERT INTO public.app_users (email, password_hash, nickname) VALUES ($1, $2, $3) RETURNING `+userFields,
		params.Email,
		params.PasswordHash,
		params.Nickname,
	)

	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO public.user_preference_profiles (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		user.ID,
	)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func (manager *UserManager) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	row := manager.db.QueryRowContext(
		ctx,
		`SELECT `+userFields+` FROM public.app_users WHERE email = $1 AND deleted_at IS NULL LIMIT 1`,
		email,
	)

	return scanUser(row)
}

func (manager *UserManager) FindByID(ctx context.Context, id string) (*model.User, error) {
	row := manager.db.QueryRowContext(
		ctx,
		`SELECT `+userFields+` FROM public.app_users WHERE id = $1 AND deleted_at IS NULL LIMIT 1`,
		id,
	)

	return scanUser(row)
}

func (manager *UserManager) UpdateLastLoginAt(ctx context.Context, id string, loginAt time.Time) error {
	_, err := manager.db.ExecContext(
		ctx,
		`UPDATE public.app_users SET last_login_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		loginAt,
		id,
	)

	return err
}

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "app_users_email_unique") || strings.Contains(message, "duplicate key value violates unique constraint")
}

func scanUser(scanner userScanner) (*model.User, error) {
	var user model.User
	var lastLoginAt sql.NullTime

	err := scanner.Scan(
		&user.ID,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.Nickname,
		&user.AvatarURL,
		&user.Gender,
		&user.Status,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	return &user, nil
}
