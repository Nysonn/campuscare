package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Log(db *pgxpool.Pool, actor uuid.UUID, action, entityType string, entityID uuid.UUID, metadata interface{}) {

	metaJSON, _ := json.Marshal(metadata)

	db.Exec(context.Background(),
		`INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, metadata)
		 VALUES ($1,$2,$3,$4,$5)`,
		actor, action, entityType, entityID, metaJSON,
	)
}
