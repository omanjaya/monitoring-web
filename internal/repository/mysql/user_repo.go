package mysql

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE id = ?`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE username = ?`

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE email = ?`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	query := `SELECT * FROM users ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, u *domain.UserCreate, passwordHash string, role string) (int64, error) {
	if role == "" {
		role = "admin"
	}
	query := `
		INSERT INTO users (username, email, password_hash, full_name, phone, role)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		u.Username, u.Email, passwordHash, u.FullName, u.Phone, role,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (r *UserRepository) Update(ctx context.Context, id int64, u *domain.UserUpdate) error {
	query := `UPDATE users SET `
	args := []interface{}{}
	setClauses := []string{}

	if u.Email != nil {
		setClauses = append(setClauses, "email = ?")
		args = append(args, *u.Email)
	}
	if u.FullName != nil {
		setClauses = append(setClauses, "full_name = ?")
		args = append(args, *u.FullName)
	}
	if u.Phone != nil {
		setClauses = append(setClauses, "phone = ?")
		args = append(args, *u.Phone)
	}
	if u.IsActive != nil {
		setClauses = append(setClauses, "is_active = ?")
		args = append(args, *u.IsActive)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query += strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, passwordHash, id)
	return err
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int64) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Keyword Repository

type KeywordRepository struct {
	db *sqlx.DB
}

func NewKeywordRepository(db *sqlx.DB) *KeywordRepository {
	return &KeywordRepository{db: db}
}

func (r *KeywordRepository) GetAll(ctx context.Context) ([]domain.Keyword, error) {
	var keywords []domain.Keyword
	query := `SELECT * FROM keywords WHERE is_active = TRUE ORDER BY weight DESC, keyword`

	err := r.db.SelectContext(ctx, &keywords, query)
	if err != nil {
		return nil, err
	}

	return keywords, nil
}

func (r *KeywordRepository) GetByCategory(ctx context.Context, category string) ([]domain.Keyword, error) {
	var keywords []domain.Keyword
	query := `SELECT * FROM keywords WHERE category = ? AND is_active = TRUE ORDER BY weight DESC`

	err := r.db.SelectContext(ctx, &keywords, query, category)
	if err != nil {
		return nil, err
	}

	return keywords, nil
}

func (r *KeywordRepository) Create(ctx context.Context, k *domain.Keyword) (int64, error) {
	query := `
		INSERT INTO keywords (keyword, category, is_regex, is_active, weight)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		k.Keyword, k.Category, k.IsRegex, k.IsActive, k.Weight,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (r *KeywordRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM keywords WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// OPD Repository

type OPDRepository struct {
	db *sqlx.DB
}

func NewOPDRepository(db *sqlx.DB) *OPDRepository {
	return &OPDRepository{db: db}
}

func (r *OPDRepository) GetAll(ctx context.Context) ([]domain.OPD, error) {
	var opds []domain.OPD
	query := `SELECT * FROM opd ORDER BY name`

	err := r.db.SelectContext(ctx, &opds, query)
	if err != nil {
		return nil, err
	}

	return opds, nil
}

func (r *OPDRepository) GetByID(ctx context.Context, id int64) (*domain.OPD, error) {
	var opd domain.OPD
	query := `SELECT * FROM opd WHERE id = ?`

	err := r.db.GetContext(ctx, &opd, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &opd, nil
}

func (r *OPDRepository) Create(ctx context.Context, o *domain.OPD) (int64, error) {
	query := `
		INSERT INTO opd (name, code, contact_email, contact_phone)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		o.Name, o.Code, o.ContactEmail, o.ContactPhone,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}
