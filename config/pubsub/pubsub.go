package pubsub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type PubSub struct {
	subs   map[TopicType][]chan string
	closed bool
}

type TopicType string

const (
	UpdateTopicType TopicType = "topic.update"
	DeleteTopicType TopicType = "topic.delete"
)

func ListenEvent(cache *redis.Client, ps *PubSub) {
	ctx := context.Background()
	updateChan := ps.Subscribe(UpdateTopicType)
	deleteChan := ps.Subscribe(DeleteTopicType)

	go handleUpdateEvents(ctx, cache, updateChan)
	go handleDeleteEvents(ctx, cache, deleteChan)
}

func handleUpdateEvents(ctx context.Context, cache *redis.Client, updateChan chan string) {
	for msg := range updateChan {
		var data UpdateEventData
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			continue
		}
		if data.Field != "" {
			cache.HSet(ctx, data.Key, data.Field, data.Data)
			cache.Expire(ctx, data.Key, data.TTL)
		} else {
			cache.Set(ctx, data.Key, data.Data, data.TTL)
		}
		log.Info().Msg("Update event processed")
	}
}

func handleDeleteEvents(ctx context.Context, cache *redis.Client, deleteChan chan string) {
	for msg := range deleteChan {
		var data DeleteEventData
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			continue
		}
		cache.Del(ctx, data.Key)
		log.Info().Str("key", data.Key).Msg("Delete event processed")
	}
}

func NewPubSub() *PubSub {
	return &PubSub{
		subs: make(map[TopicType][]chan string),
	}
}

func (ps *PubSub) Subscribe(topic TopicType) chan string {
	ch := make(chan string, 1)
	ps.subs[topic] = append(ps.subs[topic], ch)
	return ch
}

// Publish gửi message tới tất cả subscribers của một topic
func (ps *PubSub) Publish(topic TopicType, data any) {
	if ps.closed {
		return
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	for _, ch := range ps.subs[topic] {
		go func(ch chan string) {
			ch <- string(bytes)
		}(ch)
	}
}

// Close đóng tất cả channels và xóa subscribers
func (ps *PubSub) Close() {
	if !ps.closed {
		ps.closed = true
		for _, subs := range ps.subs {
			for _, ch := range subs {
				close(ch)
			}
		}
	}
}

type UpdateEventData struct {
	Key   string
	Field string
	Data  any
	TTL   time.Duration
}

type DeleteEventData struct {
	Key string
}
