package data

import (
	"context"
	"github.com/jackc/pgx/v5"
	"time"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

type PermissionsModel struct {
	DB *pgx.Conn
}

func (m PermissionsModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissons 
		INNER JOIN users_permissons ON users_permissions.permission_id = permissions.id
		INNER JOIN users_permissons ON users_permissions.user_id = users.id
		WHERE users.id = $1
	`

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (m PermissionsModel) AddForUser(userID int64, codes ...string) error {
	query := `
		INSERT INTO users_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, userID, codes)
	return err
}
