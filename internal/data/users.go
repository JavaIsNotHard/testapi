package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrRecordNotFound = errors.New("couldn't find the record")
	ErrEditConflict   = errors.New("edit conflict")
)

var AnonymousUser = &Users{}

type Users struct {
	ID        int64     `json: "id"`
	Username  string    `json: "name"`
	Email     string    `json: "email"`
	Password  password  `json: "-"`
	Activated bool      `json: "activated"`
	Version   int       `json: "-"`
	CreatedAt time.Time `json: "created_at"`
}

func (u *Users) IsAnonymous() bool {
	return u == AnonymousUser
}

func (u Users) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Username, validation.Required),
		validation.Field(&u.Email, validation.Required, is.Email),
		validation.Field(&u.Password, validation.Required),
	)
}

// validate when the users try to route to /v1/tokens/authentication
func (u Users) ValidateForAuthentication() error {
	return validation.ValidateStruct(
		validation.Field(&u.Email, validation.Required, is.Email),
		validation.Field(&u.Password, validation.Required),
	)
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintext
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintext))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil

		default:
			return false, err
		}
	}

	return true, nil
}

func (p password) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.plaintext, validation.Required.Error("Password should not be empty")),
		validation.Field(&p.hash, validation.Required.Error("Password should not be empty")),
	)
}

type UserModel struct {
	DB *pgx.Conn
}

func (m UserModel) Get(id int64) (*Users, error) {
	query := `
		SELECT id, created_at, username, email, password_hash, activated, version
		FROM users
		WHERE id = $1`

	var user Users

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (m UserModel) Insert(user *Users) error {
	query := `
		INSERT INTO users (username, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version;
	`

	args := []any{user.Username, user.Email, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateEmail
		}
	}

	return nil
}

func (m UserModel) GetUserByEmail(email string) (*Users, error) {
	query := `
		SELECT * FROM users WHERE email = $1;
	`

	var user Users

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) Update(user *Users) error {
	query := `
		UPDATE users 
		SET username = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
		WHERE id = $5 and version = $6
		RETURNING version;
	`

	args := []any{
		user.Username,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateEmail
		}
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*Users, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	query := `
		SELECT users.id, users.username, users.email, users.password_hash, users.activated, users.version, users.created_at 
		FROM users
		INNER JOIN tokens 
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1 
		AND tokens.scope = $2 
		AND tokens.expiry > $3;
	`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user Users

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
