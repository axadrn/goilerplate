package testutil

import (
	"context"
	"sync"

	"{{.ModulePath}}/internal/service/mailer"
)

type MemoryMailer struct {
	mu       sync.Mutex
	messages []mailer.Message
	err      error
}

func NewMemoryMailer() *MemoryMailer {
	return &MemoryMailer{}
}

func (m *MemoryMailer) Name() string {
	return "memory"
}

func (m *MemoryMailer) Send(_ context.Context, message mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *MemoryMailer) Messages() []mailer.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mailer.Message(nil), m.messages...)
}

func (m *MemoryMailer) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}
