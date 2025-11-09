package handlers

import (
	"sync"

	customerpb "DobrikaDev/max-bot/internal/generated/customerpb"
)

type customerStep int

const (
	customerStepNone customerStep = iota
	customerStepType
	customerStepName
	customerStepAbout
	customerStepComplete
)

type customerSession struct {
	UserID    int64
	ChatID    int64
	MessageID string
	MaxUserID string

	Type  customerpb.CustomerType
	Name  string
	About string

	Existing bool
	Current  customerStep
}

func (s *customerSession) isInProgress() bool {
	return s != nil && s.Current != customerStepNone && s.Current != customerStepComplete
}

type customerSessionStore struct {
	mu       sync.RWMutex
	sessions map[int64]*customerSession
}

func newCustomerSessionStore() *customerSessionStore {
	return &customerSessionStore{sessions: make(map[int64]*customerSession)}
}

func (s *customerSessionStore) get(userID int64) (*customerSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[userID]
	return session, ok
}

func (s *customerSessionStore) upsert(session *customerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.UserID] = session
}

func (s *customerSessionStore) delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, userID)
}
