// Package firestore is the Firestore-backed implementation of the
// agent_decision biz.Store interface. One document per decision; the
// collection name matches the old Postgres table ("agent_decisions")
// so operators reading the two generations of data see the same name.
package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"

	"j_ai_trade/modules/agent_decision/biz"
	"j_ai_trade/modules/agent_decision/model"
)

// CollectionName is the Firestore collection holding AgentDecision
// documents.
const CollectionName = "agent_decisions"

// Store writes AgentDecision documents to Firestore. The caller owns
// the *firestore.Client lifecycle (Close on shutdown).
type Store struct {
	client     *firestore.Client
	collection string
}

// NewStore wires a Store using the default collection name.
func NewStore(client *firestore.Client) *Store {
	return &Store{client: client, collection: CollectionName}
}

// Save stamps ID/timestamps if missing and writes the document. Uses
// Create so a duplicate UUID returns an error instead of silently
// overwriting.
func (s *Store) Save(ctx context.Context, d *model.AgentDecision) error {
	if d == nil {
		return nil
	}
	now := time.Now().UTC()
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now

	_, err := s.client.Collection(s.collection).Doc(d.ID.String()).Create(ctx, d)
	return err
}

var _ biz.Store = (*Store)(nil)
