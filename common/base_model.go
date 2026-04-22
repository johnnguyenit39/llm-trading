package common

import (
	"time"

	"github.com/google/uuid"
)

// BaseModel is embedded by persistence-oriented entities. ID and
// timestamps are set by stores (e.g. in-memory or external DB adapters).
//
// ID is tagged firestore:"-" because the Firestore adapter stores it as
// the document ID (see modules/agent_decision/storage/firestore). The
// raw uuid.UUID ([16]byte) would otherwise serialize as an ugly byte
// array in the document body.
type BaseModel struct {
	ID        uuid.UUID  `json:"ID" firestore:"-"`
	CreatedAt time.Time  `json:"-" firestore:"created_at"`
	UpdatedAt time.Time  `json:"-" firestore:"updated_at"`
	DeletedAt *time.Time `json:"-" firestore:"deleted_at,omitempty"`
}
