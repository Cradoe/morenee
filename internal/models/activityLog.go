package models

import "time"

type ActivityLog struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`
	Entity      string    `db:"entity"`
	EntityId    string    `db:"entity_id"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}
