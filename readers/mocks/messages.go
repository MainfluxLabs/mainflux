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

var _ readers.MessageRepository = (*messageRepositoryMock)(nil)

type messageRepositoryMock struct {
	mutex    sync.Mutex
	messages map[string][]readers.Message
}

// NewMessageRepository returns mock implementation of message repository.
func NewMessageRepository(profileID string, messages []readers.Message) readers.MessageRepository {
	repo := map[string][]readers.Message{
		profileID: messages,
	}

	return &messageRepositoryMock{
		mutex:    sync.Mutex{},
		messages: repo,
	}
}

func (repo *messageRepositoryMock) ListProfileMessages(profileID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll(profileID, rpm)
}

func (repo *messageRepositoryMock) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *messageRepositoryMock) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *messageRepositoryMock) Restore(ctx context.Context, messages ...senml.Message) error {
	panic("not implemented")
}

func (repo *messageRepositoryMock) readAll(profileID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if rpm.Format != "" && rpm.Format != "messages" {
		return readers.MessagesPage{}, nil
	}

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var msgs []readers.Message
	for _, m := range repo.messages[profileID] {
		senmlMsg := m.(senml.Message)
		if repo.messageMatchesFilter(senmlMsg, query, rpm) {
			msgs = append(msgs, m)
		}
	}

	numOfMessages := uint64(len(msgs))

	if rpm.Offset >= numOfMessages {
		return readers.MessagesPage{}, nil
	}
	if rpm.Limit < 0 {
		return readers.MessagesPage{}, nil
	}

	end := rpm.Offset + rpm.Limit
	if end > numOfMessages || rpm.Limit == noLimit {
		end = numOfMessages
	}

	return readers.MessagesPage{
		PageMetadata: rpm,
		Total:        uint64(len(msgs)),
		Messages:     msgs[rpm.Offset:end],
	}, nil
}

func (repo *messageRepositoryMock) DeleteMessages(ctx context.Context, rpm readers.PageMetadata) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	for profileID, messages := range repo.messages {
		var remainingMessages []readers.Message

		for _, m := range messages {
			senmlMsg := m.(senml.Message)

			// Keep the messages if they don't match the filter
			if !repo.messageMatchesFilter(senmlMsg, query, rpm) {
				remainingMessages = append(remainingMessages, m)
			}
		}

		repo.messages[profileID] = remainingMessages
	}

	return nil
}

func (repo *messageRepositoryMock) messageMatchesFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkFilterCondition(senmlMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) checkFilterCondition(senmlMsg senml.Message, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch filterName {
	case "subtopic":
		return rpm.Subtopic == senmlMsg.Subtopic
	case "publisher":
		return rpm.Publisher == senmlMsg.Publisher
	case "name":
		return rpm.Name == senmlMsg.Name
	case "protocol":
		return rpm.Protocol == senmlMsg.Protocol
	case "v":
		return repo.checkValueFilter(senmlMsg, query, rpm)
	case "vb":
		return repo.checkBoolValueFilter(senmlMsg, rpm)
	case "vs":
		return repo.checkStringValueFilter(senmlMsg, rpm)
	case "vd":
		return repo.checkDataValueFilter(senmlMsg, rpm)
	case "from":
		return senmlMsg.Time >= rpm.From
	case "to":
		return senmlMsg.Time < rpm.To
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkValueFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	if senmlMsg.Value == nil {
		return false
	}

	comparator, ok := query["comparator"]
	if !ok {
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
	}
}

func (repo *messageRepositoryMock) checkBoolValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.BoolValue != nil && *senmlMsg.BoolValue == rpm.BoolValue
}

func (repo *messageRepositoryMock) checkStringValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.StringValue != nil && *&senmlMsg.StringValue == &rpm.StringValue
}

func (repo *messageRepositoryMock) checkDataValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.DataValue != nil && *&senmlMsg.DataValue == &rpm.DataValue
}
