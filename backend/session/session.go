package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Role defines user permissions in a session
type Role string

const (
	RoleOwner       Role = "owner"
	RoleInteractive Role = "interactive"
	RoleReadOnly    Role = "readonly"
)

// User represents a connected client
type User struct {
	ID        string
	Role      Role
	WriteChan chan []byte
	Connected time.Time
}

// Session represents a shared terminal session
type Session struct {
	ID        string
	Owner     string
	CreatedAt time.Time
	Users     map[string]*User
	PTY       PTYHandler
	mu        sync.RWMutex
}

// PTYHandler interface for terminal operations
type PTYHandler interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
	Resize(rows, cols uint16) error
}

// Manager handles all active sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new terminal session
func (m *Manager) CreateSession(ptyHandler PTYHandler) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := uuid.New().String()[:8]
	ownerID := uuid.New().String()

	session := &Session{
		ID:        sessionID,
		Owner:     ownerID,
		CreatedAt: time.Now(),
		Users:     make(map[string]*User),
		PTY:       ptyHandler,
	}

	m.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// DeleteSession removes a session
func (m *Manager) DeleteSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		session.CloseAll()
		delete(m.sessions, id)
	}
}

// AddUser adds a user to the session
func (s *Session) AddUser(role Role) (*User, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID := uuid.New().String()
	user := &User{
		ID:        userID,
		Role:      role,
		WriteChan: make(chan []byte, 256),
		Connected: time.Now(),
	}

	s.Users[userID] = user
	return user, userID
}

// RemoveUser removes a user from the session
func (s *Session) RemoveUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user, ok := s.Users[userID]; ok {
		close(user.WriteChan)
		delete(s.Users, userID)
	}
}

// Broadcast sends data to all connected users
func (s *Session) Broadcast(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.Users {
		select {
		case user.WriteChan <- data:
		default:
			// Skip if channel is full
		}
	}
}

// Write sends input to the PTY (with role check)
func (s *Session) Write(userID string, data []byte) error {
	s.mu.RLock()
	user, ok := s.Users[userID]
	s.mu.RUnlock()

	if !ok {
		return ErrUserNotFound
	}

	if user.Role == RoleReadOnly {
		return ErrPermissionDenied
	}

	_, err := s.PTY.Write(data)
	return err
}

// GetUsers returns a list of all connected users
func (s *Session) GetUsers() []UserInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]UserInfo, 0, len(s.Users))
	for _, u := range s.Users {
		users = append(users, UserInfo{
			ID:   u.ID,
			Role: string(u.Role),
		})
	}
	return users
}

// CloseAll closes all user connections and the PTY
func (s *Session) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, user := range s.Users {
		close(user.WriteChan)
	}
	s.Users = make(map[string]*User)

	if s.PTY != nil {
		s.PTY.Close()
	}
}

// UserInfo is a serializable user representation
type UserInfo struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// Errors
var (
	ErrUserNotFound      = &Error{"user not found"}
	ErrPermissionDenied  = &Error{"permission denied"}
	ErrSessionNotFound   = &Error{"session not found"}
)

type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
