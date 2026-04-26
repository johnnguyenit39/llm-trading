package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"j_ai_trade/modules/advisor/model"
)

// fileCache is a directory-of-JSON-files cache keyed by SHA-256 of the
// full request inputs (model, temperature, seed, message array). Two
// goals:
//
//  1. Don't repay DeepSeek for the same prompt across runs. Backtest
//     iteration is exactly the kind of workload where the same 200
//     samples get replayed every time you tweak a metric, an output
//     format, or anything that doesn't change the prompt itself.
//  2. Stay zero-dep — no SQLite, no Redis. A flat directory of small
//     JSON files is git-ignorable, easy to inspect with `cat`, and
//     trivial to wipe with `rm -rf`.
//
// Hash collisions in 16 hex chars (64 bits) at 200-2000 entry scale
// are statistically zero, but full 32-byte hex would still be
// readable. We keep it short for tab-completion friendliness.
type fileCache struct {
	dir string
}

func newFileCache(dir string) (*fileCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &fileCache{dir: dir}, nil
}

// keyFor produces a stable 16-hex-char hash from the request inputs
// the LLM actually sees. Anything not on the wire (Time fields on the
// turn, internal metadata) is excluded so unrelated changes don't
// invalidate the cache.
func (c *fileCache) keyFor(modelName string, temp *float64, seed *int, turns []model.Turn) string {
	h := sha256.New()
	h.Write([]byte(modelName))
	if temp != nil {
		_ = json.NewEncoder(h).Encode(*temp)
	}
	if seed != nil {
		_ = json.NewEncoder(h).Encode(*seed)
	}
	for _, t := range turns {
		h.Write([]byte(t.Role))
		h.Write([]byte{0})
		h.Write([]byte(t.Content))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Get returns the cached response for the key, or empty + false on miss.
func (c *fileCache) Get(key string) (cachedResponse, bool) {
	path := filepath.Join(c.dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// Treat read errors like a miss — caller will refetch and
			// overwrite with a clean file.
		}
		return cachedResponse{}, false
	}
	var v cachedResponse
	if err := json.Unmarshal(data, &v); err != nil {
		return cachedResponse{}, false
	}
	return v, true
}

// Put writes the response atomically (temp file + rename) so a Ctrl-C
// mid-write can't corrupt the cache.
func (c *fileCache) Put(key string, v cachedResponse) error {
	path := filepath.Join(c.dir, key+".json")
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// cachedResponse is what we persist per key. Reply is the only field the
// caller needs; the rest is for human inspection (`cat cache/abcd.json`)
// and post-hoc cost auditing.
type cachedResponse struct {
	Reply       string  `json:"reply"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	Seed        int     `json:"seed"`
}
