package handlers

import "sync"

type menuEntry struct {
	MessageID string
	UserID    int64
}

type menuStore struct {
	mu    sync.RWMutex
	items map[int64]menuEntry
}

func newMenuStore() *menuStore {
	return &menuStore{items: make(map[int64]menuEntry)}
}

func (s *menuStore) get(chatID int64) (menuEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.items[chatID]
	return value, ok
}

func (s *menuStore) set(chatID int64, messageID string, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[chatID] = menuEntry{MessageID: messageID, UserID: userID}
}

func (s *menuStore) delete(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, chatID)
}
