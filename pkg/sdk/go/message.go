// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

const messagesEndpoint = "messages"

func (sdk mfSDK) SendMessage(subtopic, msg string, key things.ThingKey) error {
	subtopic = strings.Replace(subtopic, ".", "/", -1)
	url := fmt.Sprintf("%s/messages/%s", sdk.httpAdapterURL, subtopic)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(msg))
	if err != nil {
		return err
	}

	resp, err := sdk.sendThingRequest(req, key, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return errors.Wrap(ErrFailedPublish, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) ReadMessages(isAdmin bool, pm PageMetadata, keyType, token string) (map[string]any, error) {
	url, err := sdk.withQueryParams(sdk.readerURL, messagesEndpoint, pm)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if isAdmin {
		response, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
		if err != nil {
			return nil, err
		}

		return decodeMessages(response)
	}

	response, err := sdk.sendThingRequest(req, things.ThingKey{Value: token, Type: keyType}, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	return decodeMessages(response)
}

func (sdk mfSDK) ListJSONMessages(pm JSONPageMetadata, token string, key things.ThingKey) (map[string]any, error) {
	u := fmt.Sprintf("%s/json?%s", sdk.readerURL, pm.query())

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	if key.Value != "" {
		response, err := sdk.sendThingRequest(req, key, string(sdk.msgContentType))
		if err != nil {
			return nil, err
		}
		return decodeMessages(response)
	}

	response, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	return decodeMessages(response)
}

func (sdk mfSDK) ListSenMLMessages(pm SenMLPageMetadata, token string, key things.ThingKey) (map[string]any, error) {
	u := fmt.Sprintf("%s/senml?%s", sdk.readerURL, pm.query())

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	if key.Value != "" {
		response, err := sdk.sendThingRequest(req, key, string(sdk.msgContentType))
		if err != nil {
			return nil, err
		}
		return decodeMessages(response)
	}

	response, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	return decodeMessages(response)
}

func (sdk mfSDK) DeleteJSONMessages(publisherID, token string, pm JSONPageMetadata) error {
	u := fmt.Sprintf("%s/json/%s?%s", sdk.readerURL, publisherID, pm.query())

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) DeleteSenMLMessages(publisherID, token string, pm SenMLPageMetadata) error {
	u := fmt.Sprintf("%s/senml/%s?%s", sdk.readerURL, publisherID, pm.query())

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) DeleteAllJSONMessages(token string, pm JSONPageMetadata) error {
	u := fmt.Sprintf("%s/json?%s", sdk.readerURL, pm.query())

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) DeleteAllSenMLMessages(token string, pm SenMLPageMetadata) error {
	u := fmt.Sprintf("%s/senml?%s", sdk.readerURL, pm.query())

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) ExportJSONMessages(token string, pm JSONPageMetadata, convert, timeFormat string) ([]byte, error) {
	q := pm.query()
	if convert != "" {
		q += "&convert=" + url.QueryEscape(convert)

	}

	if timeFormat != "" {
		q += "&time_format=" + url.QueryEscape(timeFormat)
	}

	u := fmt.Sprintf("%s/json/export?%s", sdk.readerURL, q)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(ErrFailedRead, errors.New(resp.Status))
	}

	return body, nil
}

func (sdk mfSDK) ExportSenMLMessages(token string, pm SenMLPageMetadata, convert, timeFormat string) ([]byte, error) {
	q := pm.query()
	if convert != "" {
		q += "&convert=" + url.QueryEscape(convert)
	}

	if timeFormat != "" {
		q += "&time_format=" + url.QueryEscape(timeFormat)
	}

	u := fmt.Sprintf("%s/senml/export?%s", sdk.readerURL, q)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(ErrFailedRead, errors.New(resp.Status))
	}

	return body, nil
}

func (sdk mfSDK) BackupMessages(token string) (map[string]any, error) {
	u := fmt.Sprintf("%s/backup", sdk.readerURL)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	return decodeMessages(resp)
}

func (sdk mfSDK) RestoreMessages(token string, data []byte) error {
	u := fmt.Sprintf("%s/restore", sdk.readerURL)

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, "application/json")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	return nil
}

func decodeMessages(response *http.Response) (map[string]any, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(ErrFailedRead, errors.New(response.Status))
	}

	var mp map[string]any
	if err := json.Unmarshal(body, &mp); err != nil {
		return nil, err
	}

	return mp, nil
}

func (sdk mfSDK) ValidateContentType(ct ContentType) error {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return ErrInvalidContentType
	}

	return nil
}

func (pm JSONPageMetadata) query() string {
	q := url.Values{}
	q.Add("offset", strconv.FormatUint(pm.Offset, 10))
	q.Add("limit", strconv.FormatUint(pm.Limit, 10))

	if pm.Subtopic != "" {
		q.Add("subtopic", pm.Subtopic)
	}

	if pm.Publisher != "" {
		q.Add("publisher", pm.Publisher)
	}

	if pm.Protocol != "" {
		q.Add("protocol", pm.Protocol)
	}

	if pm.From != 0 {
		q.Add("from", strconv.FormatInt(pm.From, 10))
	}

	if pm.To != 0 {
		q.Add("to", strconv.FormatInt(pm.To, 10))
	}

	if pm.Filter != "" {
		q.Add("filter", pm.Filter)
	}

	if pm.AggInterval != "" {
		q.Add("agg_interval", pm.AggInterval)
	}

	if pm.AggValue != 0 {
		q.Add("agg_value", strconv.FormatUint(pm.AggValue, 10))
	}

	if pm.AggType != "" {
		q.Add("agg_type", pm.AggType)
	}

	for _, f := range pm.AggFields {
		q.Add("agg_field", f)
	}

	if pm.Dir != "" {
		q.Add("dir", pm.Dir)
	}

	return q.Encode()
}

func (pm SenMLPageMetadata) query() string {
	q := url.Values{}
	q.Add("offset", strconv.FormatUint(pm.Offset, 10))
	q.Add("limit", strconv.FormatUint(pm.Limit, 10))
	if pm.Subtopic != "" {
		q.Add("subtopic", pm.Subtopic)
	}

	if pm.Publisher != "" {
		q.Add("publisher", pm.Publisher)
	}

	if pm.Protocol != "" {
		q.Add("protocol", pm.Protocol)
	}

	if pm.Name != "" {
		q.Add("name", pm.Name)
	}

	if pm.Value != 0 {
		q.Add("v", strconv.FormatFloat(pm.Value, 'f', -1, 64))
	}

	if pm.Comparator != "" {
		q.Add("comparator", pm.Comparator)
	}

	if pm.BoolValue {
		q.Add("vb", "true")
	}

	if pm.StringValue != "" {
		q.Add("vs", pm.StringValue)
	}

	if pm.DataValue != "" {
		q.Add("vd", pm.DataValue)
	}

	if pm.From != 0 {
		q.Add("from", strconv.FormatInt(pm.From, 10))
	}

	if pm.To != 0 {
		q.Add("to", strconv.FormatInt(pm.To, 10))
	}

	if pm.AggInterval != "" {
		q.Add("agg_interval", pm.AggInterval)
	}

	if pm.AggValue != 0 {
		q.Add("agg_value", strconv.FormatUint(pm.AggValue, 10))
	}

	if pm.AggType != "" {
		q.Add("agg_type", pm.AggType)
	}

	for _, f := range pm.AggFields {
		q.Add("agg_field", f)
	}

	if pm.Dir != "" {
		q.Add("dir", pm.Dir)
	}

	return q.Encode()
}
