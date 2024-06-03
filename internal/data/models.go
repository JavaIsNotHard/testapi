package data

import (
	"github.com/jackc/pgx/v5"
)

type Models struct {
	Users       UserModel
	Permissions PermissionsModel
	Tokens      TokenModel
}

func NewModel(db *pgx.Conn) Models {
	return Models{
		Users:       UserModel{DB: db},
		Permissions: PermissionsModel{DB: db},
		Tokens:      TokenModel{DB: db},
	}
}
