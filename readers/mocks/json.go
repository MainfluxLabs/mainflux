// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/readers"
)

var _ readers.JSONMessageRepository = (*jsonRepositoryMock)(nil)

type jsonRepositoryMock struct {
	mu       sync.Mutex
	messages map[string][]readers.Message
}

func NewJSONRepository(profileID string, messages []readers.Message) readers.JSONMessageRepository {
	repo := map[string][]readers.Message{
		profileID: messages,
	}

	return &jsonRepositoryMock{
		mu:       sync.Mutex{},
		messages: repo,
	}
}

func (repo *jsonRepositoryMock) Retrieve(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return repo.readAll(rpm)
}

func (repo *jsonRepositoryMock) Backup(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return repo.readAll(rpm)
}

func (repo *jsonRepositoryMock) Remove(ctx context.Context, rpm readers.JSONPageMetadata) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for profileID, messages := range repo.messages {
		var remainingMessages []readers.Message
		for _, m := range messages {
			if !repo.messageMatchesFilter(m, rpm) {
				remainingMessages = append(remainingMessages, m)
			}
		}
		repo.messages[profileID] = remainingMessages
	}
	return nil
}

func (repo *jsonRepositoryMock) Restore(ctx context.Context, messages ...readers.Message) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}
	return nil
}

func (repo *jsonRepositoryMock) readAll(rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var filteredMessages []readers.Message

	for _, profileMessages := range repo.messages {
		for _, m := range profileMessages {
			if repo.messageMatchesFilter(m, rpm) {
				switch msg := m.(type) {
				case mfjson.Message:
					msgMap := map[string]any{
						"created":   msg.Created,
						"subtopic":  msg.Subtopic,
						"publisher": msg.Publisher,
						"protocol":  msg.Protocol,
						"payload":   msg.Payload,
					}
					filteredMessages = append(filteredMessages, msgMap)
				case map[string]any:
					filteredMessages = append(filteredMessages, msg)
				default:
					continue
				}
			}
		}
	}

	total := uint64(len(filteredMessages))

	if rpm.Offset >= total {
		return readers.JSONMessagesPage{
			JSONPageMetadata: rpm,
			MessagesPage: readers.MessagesPage{
				Total:    total,
				Messages: []readers.Message{},
			},
		}, nil
	}

	end := rpm.Offset + rpm.Limit
	if end > total || rpm.Limit == noLimit {
		end = total
	}

	return readers.JSONMessagesPage{
		JSONPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    total,
			Messages: filteredMessages[rpm.Offset:end],
		},
	}, nil
}

func (repo *jsonRepositoryMock) messageMatchesFilter(msg readers.Message, rpm readers.JSONPageMetadata) bool {
	switch m := msg.(type) {
	case mfjson.Message:
		return repo.checkJSONMessageFilter(m, rpm)
	case map[string]any:
		return repo.checkJSONMapFilter(m, rpm)
	default:
		return false
	}
}

func (repo *jsonRepositoryMock) checkJSONMessageFilter(jsonMsg mfjson.Message, rpm readers.JSONPageMetadata) bool {
	if rpm.Subtopic != "" && rpm.Subtopic != jsonMsg.Subtopic {
		return false
	}
	if rpm.Publisher != "" && rpm.Publisher != jsonMsg.Publisher {
		return false
	}
	if rpm.Protocol != "" && rpm.Protocol != jsonMsg.Protocol {
		return false
	}
	if rpm.From != 0 && jsonMsg.Created < rpm.From {
		return false
	}
	if rpm.To != 0 && jsonMsg.Created >= rpm.To {
		return false
	}
	return true
}

func (repo *jsonRepositoryMock) checkJSONMapFilter(jsonMap map[string]any, rpm readers.JSONPageMetadata) bool {
	if rpm.Subtopic != "" {
		if subtopic, ok := jsonMap["subtopic"].(string); !ok || subtopic != rpm.Subtopic {
			return false
		}
	}
	if rpm.Publisher != "" {
		if publisher, ok := jsonMap["publisher"].(string); !ok || publisher != rpm.Publisher {
			return false
		}
	}
	if rpm.Protocol != "" {
		if protocol, ok := jsonMap["protocol"].(string); !ok || protocol != rpm.Protocol {
			return false
		}
	}
	if rpm.From != 0 {
		if created, ok := jsonMap["created"].(int64); !ok || created < rpm.From {
			return false
		}
	}
	if rpm.To != 0 {
		if created, ok := jsonMap["created"].(int64); !ok || created >= rpm.From {
			return false
		}
	}
	return true
}
