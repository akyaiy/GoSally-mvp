package session

import (
	"sync"
	"time"
)

type SessionManagerContract interface {
	Add(uuid string) bool
	Delete(uuid string)
	StartCleanup(interval time.Duration)
}

type SessionManager struct {
	sessions sync.Map
	ttl      time.Duration
}

func New(ttl time.Duration) *SessionManager {
	return &SessionManager{
		ttl: ttl,
	}
}

func (sm *SessionManager) Add(uuid string) bool {
	_, loaded := sm.sessions.LoadOrStore(uuid, time.Now().Add(sm.ttl))
	return !loaded
}

func (sm *SessionManager) Delete(uuid string) {
	sm.sessions.Delete(uuid)
}

func (sm *SessionManager) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			sm.sessions.Range(func(key, value any) bool {
				expiry := value.(time.Time)
				if time.Now().After(expiry) {
					sm.sessions.Delete(key)
				}
				return true
			})
		}
	}()
}
