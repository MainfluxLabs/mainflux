// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package mocks

import (
	"context"
	"sync"

	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
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

func (repo *messageRepositoryMock) Restore(ctx context.Context, format string, messages ...readers.Message) error {
	panic("not implemented")
}

func (repo *messageRepositoryMock) readAllJSON(rpm readers.JSONMetadata) (readers.JSONMessagesPage, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var msgs []readers.Message
	for _, m := range repo.messages[""] {
		if repo.jsonMessageMatchesFilter(mfjson.Message{}, nil, rpm) {
			msgs = append(msgs, m)
		}
	}

	total := uint64(len(msgs))
	end := rpm.Offset + rpm.Limit
	if rpm.Limit == 0 || end > total {
		end = total
	}

	return readers.JSONMessagesPage{
		JSONMetadata: rpm,
		Total:        total,
		Messages:     msgs[rpm.Offset:end],
	}, nil
}

func (repo *messageRepositoryMock) deleteJSONMessages(ctx context.Context, rpm readers.JSONMetadata) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var remaining []readers.Message
	for _, m := range repo.messages[""] {
		if !repo.jsonMessageMatchesFilter(mfjson.Message{}, nil, rpm) {
			remaining = append(remaining, m)
		}
	}
	repo.messages[""] = remaining
	return nil
}

func (repo *messageRepositoryMock) readAllSenML(rpm readers.SenMLMetadata) (readers.SenMLMessagesPage, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var msgs []readers.Message
	for _, m := range repo.messages[""] {
		if repo.senmlMessageMatchesFilter(senml.Message{}, nil, rpm) {
			msgs = append(msgs, m)
		}
	}

	total := uint64(len(msgs))
	end := rpm.Offset + rpm.Limit
	if rpm.Limit == 0 || end > total {
		end = total
	}

	return readers.SenMLMessagesPage{
		SenMLMetadata: rpm,
		Total:         total,
		Messages:      msgs[rpm.Offset:end],
	}, nil
}

func (repo *messageRepositoryMock) deleteSenMLMessages(ctx context.Context, rpm readers.SenMLMetadata) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var remaining []readers.Message
	for _, m := range repo.messages[""] {
		if !repo.senmlMessageMatchesFilter(senml.Message{}, nil, rpm) {
			remaining = append(remaining, m)
		}
	}
	repo.messages[""] = remaining
	return nil
}

func (repo *messageRepositoryMock) ListJSONMessages(rpm readers.JSONMetadata) (readers.JSONMessagesPage, error) {
	return repo.readAllJSON(rpm)
}

func (repo *messageRepositoryMock) ListSenMLMessages(rpm readers.SenMLMetadata) (readers.SenMLMessagesPage, error) {
	return repo.readAllSenML(rpm)
}

func (repo *messageRepositoryMock) BackupJSONMessages(rpm readers.JSONMetadata) (readers.JSONMessagesPage, error) {
	return repo.ListJSONMessages(rpm)
}

func (repo *messageRepositoryMock) BackupSenMLMessages(rpm readers.SenMLMetadata) (readers.SenMLMessagesPage, error) {
	return repo.ListSenMLMessages(rpm)
}

func (repo *messageRepositoryMock) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}

	return nil
}

func (repo *messageRepositoryMock) RestoreSenMLMessages(ctx context.Context, messages ...readers.Message) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}

	return nil
}

func (repo *messageRepositoryMock) DeleteJSONMessages(ctx context.Context, rpm readers.JSONMetadata) error {
	return repo.deleteJSONMessages(ctx, rpm)
}

func (repo *messageRepositoryMock) DeleteSenMLMessages(ctx context.Context, rpm readers.SenMLMetadata) error {
	return repo.deleteSenMLMessages(ctx, rpm)
}

func (repo *messageRepositoryMock) senmlMessageMatchesFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.SenMLMetadata) bool {
	for name := range query {
		if !repo.checkSenmlFilterCondition(senmlMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) jsonMessageMatchesFilter(jsonMsg mfjson.Message, query map[string]interface{}, rpm readers.JSONMetadata) bool {
	for name := range query {
		if !repo.checkJsonFilterCondition(jsonMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) checkSenmlFilterCondition(senmlMsg senml.Message, filterName string, query map[string]interface{}, rpm readers.SenMLMetadata) bool {
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
		return repo.checkSenmlValueFilter(senmlMsg, query, rpm)
	case "vb":
		return repo.checkSenmlBoolValueFilter(senmlMsg, rpm)
	case "vs":
		return repo.checkSenmlStringValueFilter(senmlMsg, rpm)
	case "vd":
		return repo.checkSenmlDataValueFilter(senmlMsg, rpm)
	case "from":
		return senmlMsg.Time >= rpm.From
	case "to":
		return senmlMsg.Time < rpm.To
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkJsonFilterCondition(jsonMsg mfjson.Message, filterName string, query map[string]interface{}, rpm readers.JSONMetadata) bool {
	switch filterName {
	case "subtopic":
		return rpm.Subtopic == jsonMsg.Subtopic
	case "publisher":
		return rpm.Publisher == jsonMsg.Publisher
	case "protocol":
		return rpm.Protocol == jsonMsg.Protocol
	case "from":
		return jsonMsg.Created >= rpm.From
	case "to":
		return jsonMsg.Created < rpm.To
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkSenmlValueFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.SenMLMetadata) bool {
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
	default:
		return *senmlMsg.Value == rpm.Value
	}
}

func (repo *messageRepositoryMock) checkSenmlBoolValueFilter(senmlMsg senml.Message, rpm readers.SenMLMetadata) bool {
	return senmlMsg.BoolValue != nil && *senmlMsg.BoolValue == rpm.BoolValue
}

func (repo *messageRepositoryMock) checkSenmlStringValueFilter(senmlMsg senml.Message, rpm readers.SenMLMetadata) bool {
	return senmlMsg.StringValue != nil && *senmlMsg.StringValue == rpm.StringValue
}

func (repo *messageRepositoryMock) checkSenmlDataValueFilter(senmlMsg senml.Message, rpm readers.SenMLMetadata) bool {
	return senmlMsg.DataValue != nil && *senmlMsg.DataValue == rpm.DataValue
}
