package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jackc/pgx/v5"
)

const (
	ScopeAuthentication = "authentication"
	ScopeActivation     = "activation"
)

type Token struct {
	Token  string    `json:"token"`
	Hash   []byte    `json:"-"`
	UserID int64     `json:"-"`
	Expiry time.Time `json:"expiry"`
	Scope  string    `json:"-"`
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Token = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Token))
	token.Hash = hash[:]

	return token, nil
}

func (t Token) Validate() error {
	return validation.ValidateStruct(&t,
		validation.Field(&t.Token, validation.Required, validation.Length(26, 26)),
	)
}

type TokenModel struct {
	DB *pgx.Conn
}

func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope) 
		VALUES ($1, $2, $3, $4);
	`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, args...)
	return err
}

func (m TokenModel) DeleteForUser(scope string, userID int64) error {
	query := `
		DELETE FROM tokens 
		WHERE SCOPE = $1 AND user_id = $2;
	`

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, scope, userID)
	return err
}
