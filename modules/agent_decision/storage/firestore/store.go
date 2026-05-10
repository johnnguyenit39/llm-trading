// Package firestore is the Firestore-backed implementation of the
// agent_decision biz.Store interface. One document per decision; the
// collection name matches the old Postgres table ("agent_decisions")
// so operators reading the two generations of data see the same name.
package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

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

// PurgeOlderThan deletes every document whose created_at is before
// before. Returns the number of documents deleted. Uses batches of 400
// to stay safely under Firestore's 500-op-per-batch limit.
func (s *Store) PurgeOlderThan(ctx context.Context, before time.Time) (int, error) {
	iter := s.client.Collection(s.collection).
		Where("created_at", "<", before).
		Documents(ctx)
	defer iter.Stop()

	var deleted int
	batch := s.client.Batch()
	batchSize := 0

	flush := func() error {
		if batchSize == 0 {
			return nil
		}
		_, err := batch.Commit(ctx)
		batch = s.client.Batch()
		batchSize = 0
		return err
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return deleted, fmt.Errorf("purge query: %w", err)
		}
		batch.Delete(doc.Ref)
		batchSize++
		deleted++
		if batchSize >= 400 {
			if err := flush(); err != nil {
				return deleted, fmt.Errorf("purge commit: %w", err)
			}
		}
	}
	if err := flush(); err != nil {
		return deleted, fmt.Errorf("purge commit: %w", err)
	}
	return deleted, nil
}

var _ biz.Store = (*Store)(nil)
