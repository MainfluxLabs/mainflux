// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package mocks

import (
	"context"
	"encoding/json"
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

func (repo *messageRepositoryMock) ListProfileMessages(profileID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll(profileID, rpm)
}

func (repo *messageRepositoryMock) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *messageRepositoryMock) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *messageRepositoryMock) Restore(ctx context.Context, format string, messages ...readers.Message) error {
	panic("not implemented")
}

// ListJSONMessages implements the new interface method
func (repo *messageRepositoryMock) ListJSONMessages(rpm readers.JSONMetadata) (readers.MessagesPage, error) {
	// Convert JSONMetadata to PageMetadata for readAll
	pageMetadata := readers.PageMetadata{
		Offset:    rpm.Offset,
		Limit:     rpm.Limit,
		Subtopic:  rpm.Subtopic,
		Publisher: rpm.Publisher,
		Protocol:  rpm.Protocol,
		From:      rpm.From,
		To:        rpm.To,
	}

	// Add JSON-specific format handling
	if pageMetadata.Format == "" {
		pageMetadata.Format = "json"
	}

	page, err := repo.readAll("", pageMetadata)
	if err != nil {
		return page, err
	}

	// Update the page with JSONMetadata
	page.JSONMetadata = rpm
	return page, nil
}

// ListSenMLMessages implements the new interface method
func (repo *messageRepositoryMock) ListSenMLMessages(rpm readers.SenMLMetadata) (readers.MessagesPage, error) {
	// Convert SenMLMetadata to PageMetadata for readAll
	pageMetadata := readers.PageMetadata{
		Offset:      rpm.Offset,
		Limit:       rpm.Limit,
		Subtopic:    rpm.Subtopic,
		Publisher:   rpm.Publisher,
		Protocol:    rpm.Protocol,
		Name:        rpm.Name,
		Value:       rpm.Value,
		BoolValue:   rpm.BoolValue,
		StringValue: rpm.StringValue,
		DataValue:   rpm.DataValue,
		From:        rpm.From,
		To:          rpm.To,
		Format:      rpm.Format,
		Comparator:  rpm.Comparator,
	}

	// Add SenML-specific format handling
	if pageMetadata.Format == "" {
		pageMetadata.Format = "messages"
	}

	page, err := repo.readAll("", pageMetadata)
	if err != nil {
		return page, err
	}

	// Update the page with SenMLMetadata
	page.SenMLMetadata = rpm
	return page, nil
}

// BackupJSONMessages implements the new interface method
func (repo *messageRepositoryMock) BackupJSONMessages(rpm readers.JSONMetadata) (readers.MessagesPage, error) {
	return repo.ListJSONMessages(rpm)
}

// BackupSenMLMessages implements the new interface method
func (repo *messageRepositoryMock) BackupSenMLMessages(rpm readers.SenMLMetadata) (readers.MessagesPage, error) {
	return repo.ListSenMLMessages(rpm)
}

// RestoreJSONMessages implements the new interface method
func (repo *messageRepositoryMock) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// Add messages to the repository
	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}

	return nil
}

// RestoreSenMLMessageS implements the new interface method (note the typo in the method name matches the interface)
func (repo *messageRepositoryMock) RestoreSenMLMessageS(ctx context.Context, messages ...readers.Message) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// Add messages to the repository
	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}

	return nil
}

// DeleteJSONMessages implements the new interface method
func (repo *messageRepositoryMock) DeleteJSONMessages(ctx context.Context, rpm readers.JSONMetadata) error {
	pageMetadata := readers.PageMetadata{
		Offset:    rpm.Offset,
		Limit:     rpm.Limit,
		Subtopic:  rpm.Subtopic,
		Publisher: rpm.Publisher,
		Protocol:  rpm.Protocol,
		From:      rpm.From,
		To:        rpm.To,
		Format:    "json",
	}
	return repo.DeleteMessages(ctx, pageMetadata)
}

// DeleteSenMLMessages implements the new interface method
func (repo *messageRepositoryMock) DeleteSenMLMessages(ctx context.Context, rpm readers.SenMLMetadata) error {
	pageMetadata := readers.PageMetadata{
		Offset:      rpm.Offset,
		Limit:       rpm.Limit,
		Subtopic:    rpm.Subtopic,
		Publisher:   rpm.Publisher,
		Protocol:    rpm.Protocol,
		Name:        rpm.Name,
		Value:       rpm.Value,
		BoolValue:   rpm.BoolValue,
		StringValue: rpm.StringValue,
		DataValue:   rpm.DataValue,
		From:        rpm.From,
		To:          rpm.To,
		Format:      "messages",
		Comparator:  rpm.Comparator,
	}
	return repo.DeleteMessages(ctx, pageMetadata)
}

func (repo *messageRepositoryMock) readAll(profileID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if rpm.Format != "" && rpm.Format != "messages" && rpm.Format != "json" {
		return readers.MessagesPage{}, nil
	}

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var msgs []readers.Message
	for _, m := range repo.messages[profileID] {
		if repo.messageMatchesFilter(m, query, rpm) {
			if msgMap, ok := m.(map[string]interface{}); ok && rpm.Format == "json" {
				jsonMsg := repo.mapToJSONMessage(msgMap)
				msgs = append(msgs, jsonMsg)
			} else {
				msgs = append(msgs, m)
			}
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

	var messagesPage readers.MessagesPage

	switch rpm.Format {
	case "json":
		jsonMeta := readers.JSONMetadata{
			Offset:    rpm.Offset,
			Limit:     rpm.Limit,
			Subtopic:  rpm.Subtopic,
			Publisher: rpm.Publisher,
			Protocol:  rpm.Protocol,
			From:      rpm.From,
			To:        rpm.To,
		}
		messagesPage = readers.MessagesPage{
			JSONMetadata: jsonMeta,
			Total:        uint64(len(msgs)),
			Messages:     msgs[rpm.Offset:end],
		}
	default:
		senmlMeta := readers.SenMLMetadata{
			Offset:      rpm.Offset,
			Limit:       rpm.Limit,
			Subtopic:    rpm.Subtopic,
			Publisher:   rpm.Publisher,
			Protocol:    rpm.Protocol,
			Name:        rpm.Name,
			Value:       rpm.Value,
			BoolValue:   rpm.BoolValue,
			StringValue: rpm.StringValue,
			DataValue:   rpm.DataValue,
			From:        rpm.From,
			To:          rpm.To,
			Format:      "messages",
			Comparator:  rpm.Comparator,
		}

		messagesPage = readers.MessagesPage{
			SenMLMetadata: senmlMeta,
			Total:         uint64(len(msgs)),
			Messages:      msgs[rpm.Offset:end],
		}
	}

	return messagesPage, nil
}

func (repo *messageRepositoryMock) mapToJSONMessage(msgMap map[string]interface{}) mfjson.Message {
	msg := mfjson.Message{}
	if created, ok := msgMap["created"].(int64); ok {
		msg.Created = created
	}

	if subtopic, ok := msgMap["subtopic"].(string); ok {
		msg.Subtopic = subtopic
	}

	if publisher, ok := msgMap["publisher"].(string); ok {
		msg.Publisher = publisher
	}

	if protocol, ok := msgMap["protocol"].(string); ok {
		msg.Protocol = protocol
	}

	if payload, ok := msgMap["payload"].(map[string]interface{}); ok {
		payloadBytes, _ := json.Marshal(payload)
		msg.Payload = payloadBytes
	}

	return msg
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
			// Keep the messages if they don't match the filter
			if !repo.messageMatchesFilter(m, query, rpm) {
				remainingMessages = append(remainingMessages, m)
			}
		}

		repo.messages[profileID] = remainingMessages
	}

	return nil
}

func (repo *messageRepositoryMock) messageMatchesFilter(msg readers.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch m := msg.(type) {
	case senml.Message:
		return repo.senmlMessageMatchesFilter(m, query, rpm)
	case mfjson.Message:
		return repo.jsonMessageMatchesFilter(m, query, rpm)
	case map[string]interface{}:
		return repo.jsonMapMessageMatchesFilter(m, query, rpm)
	default:
		return false
	}
}

func (repo *messageRepositoryMock) senmlMessageMatchesFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkSenmlFilterCondition(senmlMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) jsonMessageMatchesFilter(jsonMsg mfjson.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkJsonFilterCondition(jsonMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) jsonMapMessageMatchesFilter(jsonMap map[string]interface{}, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkJsonMapFilterCondition(jsonMap, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) checkJsonMapFilterCondition(jsonMap map[string]interface{}, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch filterName {
	case "subtopic":
		if subtopic, ok := jsonMap["subtopic"].(string); ok {
			return rpm.Subtopic == subtopic
		}
		return rpm.Subtopic == ""
	case "publisher":
		if publisher, ok := jsonMap["publisher"].(string); ok {
			return rpm.Publisher == publisher
		}
		return rpm.Publisher == ""
	case "protocol":
		if protocol, ok := jsonMap["protocol"].(string); ok {
			return rpm.Protocol == protocol
		}
		return rpm.Protocol == ""
	case "from":
		if created, ok := jsonMap["created"].(float64); ok {
			return int64(created) >= rpm.From
		}
		if created, ok := jsonMap["created"].(int64); ok {
			return created >= rpm.From
		}
		return false
	case "to":
		if created, ok := jsonMap["created"].(float64); ok {
			return int64(created) < rpm.To
		}
		if created, ok := jsonMap["created"].(int64); ok {
			return created < rpm.To
		}
		return false
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkSenmlFilterCondition(senmlMsg senml.Message, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
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

func (repo *messageRepositoryMock) checkJsonFilterCondition(jsonMsg mfjson.Message, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
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

func (repo *messageRepositoryMock) checkSenmlValueFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
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

func (repo *messageRepositoryMock) checkSenmlBoolValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.BoolValue != nil && *senmlMsg.BoolValue == rpm.BoolValue
}

func (repo *messageRepositoryMock) checkSenmlStringValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.StringValue != nil && *senmlMsg.StringValue == rpm.StringValue
}

func (repo *messageRepositoryMock) checkSenmlDataValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.DataValue != nil && *senmlMsg.DataValue == rpm.DataValue
}
