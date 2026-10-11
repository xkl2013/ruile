package stream

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// memoryStreamData holds stream events in memory
type memoryStreamData struct {
	events      []interfaces.StreamEvent
	steerEvents []interfaces.StreamEvent
	lastUpdated time.Time
	mu          sync.RWMutex
}

var ErrLiveRunExists = errors.New("session already has a live run")

type liveRunMarker struct {
	assistantMessageID string
	requestID          string
}

// MemoryStreamManager implements StreamManager using in-memory storage
type MemoryStreamManager struct {
	// Map: sessionID -> messageID -> stream data
	streams  map[string]map[string]*memoryStreamData
	liveRuns map[string]liveRunMarker
	mu       sync.RWMutex
}

// NewMemoryStreamManager creates a new in-memory stream manager
func NewMemoryStreamManager() *MemoryStreamManager {
	return &MemoryStreamManager{
		streams:  make(map[string]map[string]*memoryStreamData),
		liveRuns: make(map[string]liveRunMarker),
	}
}

// getOrCreateStream gets or creates stream data
func (m *MemoryStreamManager) getOrCreateStream(sessionID, messageID string) *memoryStreamData {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streams[sessionID]; !exists {
		m.streams[sessionID] = make(map[string]*memoryStreamData)
	}

	if _, exists := m.streams[sessionID][messageID]; !exists {
		m.streams[sessionID][messageID] = &memoryStreamData{
			events:      make([]interfaces.StreamEvent, 0),
			lastUpdated: time.Now(),
		}
	}

	return m.streams[sessionID][messageID]
}

// getStream gets existing stream data (returns nil if not found)
func (m *MemoryStreamManager) getStream(sessionID, messageID string) *memoryStreamData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sessionMap, exists := m.streams[sessionID]; exists {
		return sessionMap[messageID]
	}
	return nil
}

// AppendEvent appends a single event to the stream
func (m *MemoryStreamManager) AppendEvent(
	ctx context.Context,
	sessionID, messageID string,
	event interfaces.StreamEvent,
) error {
	stream := m.getOrCreateStream(sessionID, messageID)

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Append event
	stream.events = append(stream.events, event)
	stream.lastUpdated = time.Now()

	return nil
}

// GetEvents gets events starting from offset
// Returns: events slice, next offset, error
func (m *MemoryStreamManager) GetEvents(
	ctx context.Context,
	sessionID, messageID string,
	fromOffset int,
) ([]interfaces.StreamEvent, int, error) {
	stream := m.getStream(sessionID, messageID)
	if stream == nil {
		// Stream doesn't exist yet
		return []interfaces.StreamEvent{}, fromOffset, nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	// Check if offset is beyond current events
	if fromOffset >= len(stream.events) {
		return []interfaces.StreamEvent{}, fromOffset, nil
	}

	// Get events from offset to end
	events := stream.events[fromOffset:]
	nextOffset := len(stream.events)

	// Return copy of events to avoid race conditions
	eventsCopy := make([]interfaces.StreamEvent, len(events))
	copy(eventsCopy, events)

	return eventsCopy, nextOffset, nil
}

func (m *MemoryStreamManager) AppendSteerEvents(
	_ context.Context, sessionID, messageID string, events []interfaces.StreamEvent,
) error {
	stream := m.getOrCreateStream(sessionID, messageID)
	stream.mu.Lock()
	defer stream.mu.Unlock()

	seen := make(map[string]struct{}, len(stream.steerEvents))
	for _, existing := range stream.steerEvents {
		seen[existing.ID] = struct{}{}
	}
	for _, event := range events {
		if event.ID != "" {
			if _, ok := seen[event.ID]; ok {
				continue
			}
			seen[event.ID] = struct{}{}
		}
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}
		stream.steerEvents = append(stream.steerEvents, event)
	}
	stream.lastUpdated = time.Now()
	return nil
}

func (m *MemoryStreamManager) GetSteerEvents(
	_ context.Context, sessionID, messageID string, fromOffset int,
) ([]interfaces.StreamEvent, int, error) {
	stream := m.getStream(sessionID, messageID)
	if stream == nil {
		return []interfaces.StreamEvent{}, fromOffset, nil
	}
	stream.mu.RLock()
	defer stream.mu.RUnlock()
	if fromOffset >= len(stream.steerEvents) {
		return []interfaces.StreamEvent{}, fromOffset, nil
	}
	events := append([]interfaces.StreamEvent(nil), stream.steerEvents[fromOffset:]...)
	return events, len(stream.steerEvents), nil
}

func (m *MemoryStreamManager) UpdateSteerEventData(
	_ context.Context, sessionID, messageID, eventID string, data map[string]interface{},
) (bool, error) {
	stream := m.getStream(sessionID, messageID)
	if stream == nil {
		return false, nil
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	for i := range stream.steerEvents {
		if stream.steerEvents[i].ID != eventID {
			continue
		}
		merged := make(map[string]interface{}, len(stream.steerEvents[i].Data)+len(data))
		for key, value := range stream.steerEvents[i].Data {
			merged[key] = value
		}
		for key, value := range data {
			merged[key] = value
		}
		stream.steerEvents[i].Data = merged
		stream.lastUpdated = time.Now()
		return true, nil
	}
	return false, nil
}

func (m *MemoryStreamManager) DeleteSteerEvent(
	_ context.Context, sessionID, messageID, eventID string,
) (bool, error) {
	stream := m.getStream(sessionID, messageID)
	if stream == nil {
		return false, nil
	}
	stream.mu.Lock()
	defer stream.mu.Unlock()
	for i := range stream.steerEvents {
		if stream.steerEvents[i].ID != eventID {
			continue
		}
		if consumed, _ := stream.steerEvents[i].Data["consumed"].(bool); consumed {
			return false, nil
		}
		stream.steerEvents = append(stream.steerEvents[:i], stream.steerEvents[i+1:]...)
		stream.lastUpdated = time.Now()
		return true, nil
	}
	return false, nil
}

func (m *MemoryStreamManager) SetLiveRun(
	_ context.Context, sessionID, assistantMessageID, requestID string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.liveRuns[sessionID]; ok && existing.assistantMessageID != assistantMessageID {
		return ErrLiveRunExists
	}
	m.liveRuns[sessionID] = liveRunMarker{assistantMessageID: assistantMessageID, requestID: requestID}
	return nil
}

func (m *MemoryStreamManager) ClaimLiveRun(
	_ context.Context, sessionID, assistantMessageID, requestID string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.liveRuns[sessionID] = liveRunMarker{assistantMessageID: assistantMessageID, requestID: requestID}
	return nil
}

func (m *MemoryStreamManager) GetLiveRun(
	_ context.Context, sessionID string,
) (string, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	marker, ok := m.liveRuns[sessionID]
	if !ok {
		return "", "", nil
	}
	return marker.assistantMessageID, marker.requestID, nil
}

func (m *MemoryStreamManager) ClearLiveRun(
	_ context.Context, sessionID, assistantMessageID string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if marker, ok := m.liveRuns[sessionID]; ok && marker.assistantMessageID == assistantMessageID {
		delete(m.liveRuns, sessionID)
	}
	return nil
}

// Ensure MemoryStreamManager implements StreamManager interface
var _ interfaces.StreamManager = (*MemoryStreamManager)(nil)
