// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
)

const noLimit = 0

var _ readers.SenMLMessageRepository = (*senmlRepositoryMock)(nil)

type senmlRepositoryMock struct {
	mu       sync.Mutex
	messages map[string][]readers.Message
}

func NewSenMLRepository(profileID string, messages []readers.Message) readers.SenMLMessageRepository {
	repo := map[string][]readers.Message{
		profileID: messages,
	}

	return &senmlRepositoryMock{
		mu:       sync.Mutex{},
		messages: repo,
	}
}

func (repo *senmlRepositoryMock) Retrieve(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return repo.readAll(rpm)
}

func (repo *senmlRepositoryMock) Backup(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return repo.readAll(rpm)
}

func (repo *senmlRepositoryMock) Restore(ctx context.Context, messages ...readers.Message) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}
	return nil
}

func (repo *senmlRepositoryMock) Remove(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var query map[string]any
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	for profileID, messages := range repo.messages {
		var remainingMessages []readers.Message
		for _, m := range messages {
			if !repo.senmlMessageMatchesFilter(m, query, rpm) {
				remainingMessages = append(remainingMessages, m)
			}
		}
		repo.messages[profileID] = remainingMessages
	}
	return nil
}

func (repo *senmlRepositoryMock) readAll(rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var query map[string]any
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var msgs []readers.Message
	for _, profileMessages := range repo.messages {
		for _, m := range profileMessages {
			if repo.senmlMessageMatchesFilter(m, query, rpm) {
				msgs = append(msgs, m)
			}
		}
	}

	total := uint64(len(msgs))

	if rpm.Offset >= total {
		return readers.SenMLMessagesPage{
			SenMLPageMetadata: rpm,
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

	return readers.SenMLMessagesPage{
		SenMLPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    total,
			Messages: msgs[rpm.Offset:end],
		},
	}, nil
}

func (repo *senmlRepositoryMock) senmlMessageMatchesFilter(msg readers.Message, query map[string]any, rpm readers.SenMLPageMetadata) bool {
	switch m := msg.(type) {
	case senml.Message:
		return repo.checkSenMLMessageFilter(m, query, rpm)
	default:
		return false
	}
}

func (repo *senmlRepositoryMock) checkSenMLMessageFilter(senmlMsg senml.Message, query map[string]any, rpm readers.SenMLPageMetadata) bool {
	if rpm.Subtopic != "" && rpm.Subtopic != senmlMsg.Subtopic {
		return false
	}
	if rpm.Publisher != "" && rpm.Publisher != senmlMsg.Publisher {
		return false
	}
	if rpm.Protocol != "" && rpm.Protocol != senmlMsg.Protocol {
		return false
	}
	if rpm.Name != "" && rpm.Name != senmlMsg.Name {
		return false
	}
	if rpm.From != 0 && senmlMsg.Time < rpm.From {
		return false
	}
	if rpm.To != 0 && senmlMsg.Time >= rpm.To {
		return false
	}

	if !repo.checkSenMLValueFilters(senmlMsg, query, rpm) {
		return false
	}

	return true
}

func (repo *senmlRepositoryMock) checkSenMLValueFilters(senmlMsg senml.Message, query map[string]any, rpm readers.SenMLPageMetadata) bool {
	if _, hasValue := query["v"]; hasValue && senmlMsg.Value != nil {
		comparator, hasComparator := query["comparator"]
		if !hasComparator {
			return *senmlMsg.Value == rpm.Value
		}

		switch comparator.(string) {
		case readers.LowerThanKey:
			return *senmlMsg.Value < rpm.Value
		case readers.LowerThanEqualKey:
			return *senmlMsg.Value <= rpm.Value
		case readers.GreaterThanKey:
			return *senmlMsg.Value > rpm.Value
		case readers.GreaterThanEqualKey:
			return *senmlMsg.Value >= rpm.Value
		case readers.EqualKey:
			return *senmlMsg.Value == rpm.Value
		default:
			return *senmlMsg.Value == rpm.Value
		}
	}

	if _, hasBool := query["vb"]; hasBool && senmlMsg.BoolValue != nil {
		return *senmlMsg.BoolValue == rpm.BoolValue
	}

	if _, hasString := query["vs"]; hasString && senmlMsg.StringValue != nil {
		return *senmlMsg.StringValue == rpm.StringValue
	}

	if _, hasData := query["vd"]; hasData && senmlMsg.DataValue != nil {
		return *senmlMsg.DataValue == rpm.DataValue
	}

	if _, hasValue := query["v"]; !hasValue {
		if _, hasBool := query["vb"]; !hasBool {
			if _, hasString := query["vs"]; !hasString {
				if _, hasData := query["vd"]; !hasData {
					return true
				}
			}
		}
	}

	return false
}
