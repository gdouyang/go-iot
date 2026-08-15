package agent

import (
	"sort"
	"sync"

	"go-iot/pkg/models"
)

// Store persists Agent entities. Tests use MemoryStore; production can swap ES.
type Store interface {
	SaveConversation(c *models.AgentConversation) error
	GetConversation(id string) (*models.AgentConversation, error)
	ListConversations(userId int64) ([]models.AgentConversation, error)
	DeleteConversation(id string) error

	SaveMessage(m *models.AgentMessage) error
	ListMessages(convId string) ([]models.AgentMessage, error)

	SaveDraft(d *models.AgentDraft) error
	GetDraft(id string) (*models.AgentDraft, error)
	ListDrafts(convId string) ([]models.AgentDraft, error)

	SaveAudit(a *models.AgentAudit) error

	GetSettings(userId int64) (*models.AgentUserSettings, error)
	SaveSettings(s *models.AgentUserSettings) error
}

// MemoryStore is the test/default in-process store (shipped, used by HTTP tests).
type MemoryStore struct {
	mu    sync.RWMutex
	convs map[string]*models.AgentConversation
	msgs  map[string][]models.AgentMessage
	drafts map[string]*models.AgentDraft
	audits []models.AgentAudit
	sets  map[int64]*models.AgentUserSettings
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		convs:  map[string]*models.AgentConversation{},
		msgs:   map[string][]models.AgentMessage{},
		drafts: map[string]*models.AgentDraft{},
		sets:   map[int64]*models.AgentUserSettings{},
	}
}

func (s *MemoryStore) SaveConversation(c *models.AgentConversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.UpdateTime = models.NewDateTime()
	cp := *c
	s.convs[c.Id] = &cp
	return nil
}

func (s *MemoryStore) GetConversation(id string) (*models.AgentConversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.convs[id]
	if c == nil {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (s *MemoryStore) ListConversations(userId int64) ([]models.AgentConversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.AgentConversation
	for _, c := range s.convs {
		if c.CreateId == userId {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ti, tj := out[i].UpdateTime.UnixMilli(), out[j].UpdateTime.UnixMilli()
		if ti != tj {
			return ti > tj
		}
		return out[i].Id > out[j].Id
	})
	return out, nil
}

func (s *MemoryStore) DeleteConversation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.convs, id)
	delete(s.msgs, id)
	for k, d := range s.drafts {
		if d.ConversationId == id {
			delete(s.drafts, k)
		}
	}
	return nil
}

func (s *MemoryStore) SaveMessage(m *models.AgentMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	StampMessage(m)
	s.msgs[m.ConversationId] = append(s.msgs[m.ConversationId], *m)
	return nil
}

func (s *MemoryStore) ListMessages(convId string) ([]models.AgentMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.msgs[convId]
	out := make([]models.AgentMessage, len(src))
	copy(out, src)
	SortMessages(out)
	return out, nil
}

func (s *MemoryStore) SaveDraft(d *models.AgentDraft) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *d
	s.drafts[d.Id] = &cp
	return nil
}

func (s *MemoryStore) GetDraft(id string) (*models.AgentDraft, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := s.drafts[id]
	if d == nil {
		return nil, nil
	}
	cp := *d
	return &cp, nil
}

func (s *MemoryStore) ListDrafts(convId string) ([]models.AgentDraft, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.AgentDraft
	for _, d := range s.drafts {
		if d.ConversationId == convId {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveAudit(a *models.AgentAudit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, *a)
	return nil
}

func (s *MemoryStore) GetSettings(userId int64) (*models.AgentUserSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.sets[userId]
	if st == nil {
		return nil, nil
	}
	cp := *st
	return &cp, nil
}

func (s *MemoryStore) SaveSettings(st *models.AgentUserSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *st
	s.sets[st.UserId] = &cp
	return nil
}

// DefaultStore is used by HTTP handlers. Tests replace it.
var DefaultStore Store = NewMemoryStore()
