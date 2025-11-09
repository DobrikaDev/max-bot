package handlers

import (
	"fmt"
	"sync"

	userpb "DobrikaDev/max-bot/internal/generated/userpb"
)

type registrationStep int

const (
	registrationStepNone registrationStep = iota
	registrationStepAge
	registrationStepSex
	registrationStepLocation
	registrationStepAbout
	registrationStepComplete
)

type registrationSession struct {
	UserID    int64
	ChatID    int64
	UserName  string
	MaxUserID string

	Age                       int32
	Sex                       userpb.Sex
	Latitude                  float64
	Longitude                 float64
	GeoLabel                  string
	About                     string
	Interests                 map[int]bool
	OriginalAboutOptionPrefix map[int]string
	Current                   registrationStep
	MessageID                 string
}

func (s *registrationSession) geolocationAsString() string {
	switch {
	case s.Latitude != 0 || s.Longitude != 0:
		return fmt.Sprintf("%f,%f", s.Latitude, s.Longitude)
	case s.GeoLabel != "":
		return s.GeoLabel
	default:
		return ""
	}
}

func (s *registrationSession) isInProgress() bool {
	return s != nil && s.Current != registrationStepNone && s.Current != registrationStepComplete
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[int64]*registrationSession
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[int64]*registrationSession)}
}

func (s *sessionStore) get(userID int64) (*registrationSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[userID]
	return session, ok
}

func (s *sessionStore) upsert(session *registrationSession) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.UserID] = session
}

func (s *sessionStore) delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, userID)
}
