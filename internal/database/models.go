// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package database

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Author struct {
	ID       int64       `json:"id"`
	Name     string      `json:"name"`
	Bio      pgtype.Text `json:"bio"`
	BookName pgtype.Text `json:"bookName"`
}
