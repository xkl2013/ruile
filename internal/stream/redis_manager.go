package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tencent/WeKnora/internal/common"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/redis/go-redis/v9"
)

// RedisStreamManager implements StreamManager using Redis Lists for append-only event streaming
type RedisStreamManager struct {
	client *redis.Client
	ttl    time.Duration // TTL for stream data in Redis
	prefix string        // Redis key prefix
}

// NewRedisStreamManager creates a new Redis-based stream manager
func NewRedisStreamManager(redisAddr, redisUsername, redisPassword string,
	redisDB int, prefix string, ttl time.Duration,
) (*RedisStreamManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:      redisAddr,
		Username:  redisUsername,
		Password:  redisPassword,
		DB:        redisDB,
		TLSConfig: common.RedisTLSConfig(),
	})

	// Verify connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	if ttl == 0 {
		ttl = 24 * time.Hour // Default TTL: 24 hours
	}

	if prefix == "" {
		prefix = "stream:events" // Default prefix
	}

	return &RedisStreamManager{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}, nil
}

// buildKey builds the Redis key for event list
func (r *RedisStreamManager) buildKey(sessionID, messageID string) string {
	return fmt.Sprintf("%s:%s:%s", r.prefix, sessionID, messageID)
}

func (r *RedisStreamManager) buildSteerKey(sessionID, messageID string) string {
	return fmt.Sprintf("%s:%s:%s:steer", r.prefix, sessionID, messageID)
}

func (r *RedisStreamManager) buildLiveRunKey(sessionID string) string {
	return fmt.Sprintf("%s:%s:live-run", r.prefix, sessionID)
}

// AppendEvent appends a single event to the stream using Redis RPush
func (r *RedisStreamManager) AppendEvent(
	ctx context.Context,
	sessionID, messageID string,
	event interfaces.StreamEvent,
) error {
	key := r.buildKey(sessionID, messageID)

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Serialize event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Append to Redis list with RPush (O(1) operation)
	if err := r.client.RPush(ctx, key, eventJSON).Err(); err != nil {
		return fmt.Errorf("failed to append event to Redis: %w", err)
	}

	// Set/refresh TTL on the key
	if err := r.client.Expire(ctx, key, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	return nil
}

// GetEvents gets events starting from offset using Redis LRange
// Returns: events slice, next offset, error
func (r *RedisStreamManager) GetEvents(
	ctx context.Context,
	sessionID, messageID string,
	fromOffset int,
) ([]interfaces.StreamEvent, int, error) {
	key := r.buildKey(sessionID, messageID)

	// Get all events from offset to end using LRange
	// LRange is inclusive, so fromOffset to -1 gets all remaining elements
	results, err := r.client.LRange(ctx, key, int64(fromOffset), -1).Result()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist - return empty slice
			return []interfaces.StreamEvent{}, fromOffset, nil
		}
		return nil, fromOffset, fmt.Errorf("failed to get events from Redis: %w", err)
	}
	r.touchLiveRun(ctx, sessionID)

	// No new events
	if len(results) == 0 {
		return []interfaces.StreamEvent{}, fromOffset, nil
	}

	// Unmarshal events
	events := make([]interfaces.StreamEvent, 0, len(results))
	for _, result := range results {
		var event interfaces.StreamEvent
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			// Log error but continue with other events
			continue
		}
		events = append(events, event)
	}

	// Calculate next offset
	nextOffset := fromOffset + len(results)

	return events, nextOffset, nil
}

func (r *RedisStreamManager) touchLiveRun(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	_, _ = r.client.Expire(ctx, r.buildLiveRunKey(sessionID), r.ttl).Result()
}

func (r *RedisStreamManager) AppendSteerEvents(
	ctx context.Context, sessionID, messageID string, events []interfaces.StreamEvent,
) error {
	if len(events) == 0 {
		return nil
	}
	key := r.buildSteerKey(sessionID, messageID)
	payloads := make([]interface{}, 0, len(events))
	for _, event := range events {
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal steer event: %w", err)
		}
		payloads = append(payloads, payload)
	}
	// De-duplication must be atomic: a timed-out request can be retried while
	// another API replica is appending the same client-generated steer ID.
	args := append([]interface{}{int64(r.ttl / time.Second)}, payloads...)
	if err := steerAppendUnique.Run(ctx, r.client, []string{key}, args...).Err(); err != nil {
		return fmt.Errorf("failed to append steer events to Redis: %w", err)
	}
	_ = r.client.Expire(ctx, r.buildLiveRunKey(sessionID), r.ttl).Err()
	return nil
}

var steerAppendUnique = redis.NewScript(`
local seen = {}
for _, raw in ipairs(redis.call('LRANGE', KEYS[1], 0, -1)) do
  local ok, event = pcall(cjson.decode, raw)
  if ok and event.id then seen[event.id] = true end
end
for i = 2, #ARGV do
  local event = cjson.decode(ARGV[i])
  if not event.id or event.id == '' or not seen[event.id] then
    redis.call('RPUSH', KEYS[1], ARGV[i])
    if event.id then seen[event.id] = true end
  end
end
redis.call('EXPIRE', KEYS[1], ARGV[1])
return 1
`)

func (r *RedisStreamManager) GetSteerEvents(
	ctx context.Context, sessionID, messageID string, fromOffset int,
) ([]interfaces.StreamEvent, int, error) {
	key := r.buildSteerKey(sessionID, messageID)
	results, err := r.client.LRange(ctx, key, int64(fromOffset), -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []interfaces.StreamEvent{}, fromOffset, nil
		}
		return nil, fromOffset, fmt.Errorf("failed to get steer events: %w", err)
	}
	r.touchLiveRun(ctx, sessionID)
	events := make([]interfaces.StreamEvent, 0, len(results))
	for _, result := range results {
		var event interfaces.StreamEvent
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, fromOffset + len(results), nil
}

func (r *RedisStreamManager) UpdateSteerEventData(
	ctx context.Context, sessionID, messageID, eventID string, data map[string]interface{},
) (bool, error) {
	key := r.buildSteerKey(sessionID, messageID)
	results, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to read steer events: %w", err)
	}
	for index, result := range results {
		var event interfaces.StreamEvent
		if json.Unmarshal([]byte(result), &event) != nil || event.ID != eventID {
			continue
		}
		if event.Data == nil {
			event.Data = map[string]interface{}{}
		}
		for key, value := range data {
			event.Data[key] = value
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return false, fmt.Errorf("failed to marshal steer event: %w", err)
		}
		if err := r.client.LSet(ctx, key, int64(index), payload).Err(); err != nil {
			return false, fmt.Errorf("failed to update steer event: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func (r *RedisStreamManager) DeleteSteerEvent(
	ctx context.Context, sessionID, messageID, eventID string,
) (bool, error) {
	key := r.buildSteerKey(sessionID, messageID)
	results, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to read steer events: %w", err)
	}
	for _, result := range results {
		var event interfaces.StreamEvent
		if json.Unmarshal([]byte(result), &event) != nil || event.ID != eventID {
			continue
		}
		if consumed, _ := event.Data["consumed"].(bool); consumed {
			return false, nil
		}
		removed, err := r.client.LRem(ctx, key, 1, result).Result()
		if err != nil {
			return false, fmt.Errorf("failed to delete steer event: %w", err)
		}
		return removed > 0, nil
	}
	return false, nil
}

type redisLiveRunPayload struct {
	AssistantMessageID string `json:"assistant_message_id"`
	RequestID          string `json:"request_id"`
}

func (r *RedisStreamManager) SetLiveRun(
	ctx context.Context, sessionID, assistantMessageID, requestID string,
) error {
	payload, err := json.Marshal(redisLiveRunPayload{AssistantMessageID: assistantMessageID, RequestID: requestID})
	if err != nil {
		return err
	}
	key := r.buildLiveRunKey(sessionID)
	ok, err := r.client.SetNX(ctx, key, payload, r.ttl).Result()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	current, _, err := r.GetLiveRun(ctx, sessionID)
	if err != nil {
		return err
	}
	if current == assistantMessageID {
		return nil
	}
	return ErrLiveRunExists
}

func (r *RedisStreamManager) ClaimLiveRun(
	ctx context.Context, sessionID, assistantMessageID, requestID string,
) error {
	payload, err := json.Marshal(redisLiveRunPayload{AssistantMessageID: assistantMessageID, RequestID: requestID})
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.buildLiveRunKey(sessionID), payload, r.ttl).Err()
}

func (r *RedisStreamManager) GetLiveRun(
	ctx context.Context, sessionID string,
) (string, string, error) {
	value, err := r.client.Get(ctx, r.buildLiveRunKey(sessionID)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", "", nil
		}
		return "", "", err
	}
	var payload redisLiveRunPayload
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return "", "", err
	}
	return payload.AssistantMessageID, payload.RequestID, nil
}

func (r *RedisStreamManager) ClearLiveRun(
	ctx context.Context, sessionID, assistantMessageID string,
) error {
	current, _, err := r.GetLiveRun(ctx, sessionID)
	if err != nil || current != assistantMessageID {
		return err
	}
	return r.client.Del(ctx, r.buildLiveRunKey(sessionID)).Err()
}

// Close closes the Redis connection
func (r *RedisStreamManager) Close() error {
	return r.client.Close()
}

// Ensure RedisStreamManager implements StreamManager interface
var _ interfaces.StreamManager = (*RedisStreamManager)(nil)
