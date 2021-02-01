//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package re

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	SDK "github.com/mainflux/mainflux/pkg/sdk/go"
)

const (
	host = "http://localhost:9081"
)

var (
	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrKuiperServer indicates internal kuiper rules engine server error
	ErrKuiperServer = errors.New("kuiper internal server error")
)

type Info struct {
	Version       string `json:"version"`
	Os            string `json:"os"`
	UpTimeSeconds int    `json:"upTimeSeconds"`
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	Info(ctx context.Context) (Info, error)
	CreateStream(ctx context.Context, token, name, topic, row string, update bool) (string, error)
	ListStreams(ctx context.Context, token string) ([]string, error)
	ViewStream(ctx context.Context, token, name string) (Stream, error)
	Delete(ctx context.Context, token, name, kind string) (string, error)

	CreateRule(ctx context.Context, token string, rule Rule, update bool) (string, error)
	ListRules(ctx context.Context, token string) ([]RuleInfo, error)
	ViewRule(ctx context.Context, token, name string) (Rule, error)
	GetRuleStatus(ctx context.Context, token, name string) (map[string]interface{}, error)
	ControlRule(ctx context.Context, token, name, action string) (string, error)
}

type reService struct {
	auth   mainflux.AuthServiceClient
	sdk    SDK.SDK
	logger logger.Logger
}

var _ Service = (*reService)(nil)

// New instantiates the re service implementation.
func New(auth mainflux.AuthServiceClient, sdk SDK.SDK, logger logger.Logger) Service {
	return &reService{
		auth:   auth,
		sdk:    sdk,
		logger: logger,
	}
}

func (re *reService) Info(_ context.Context) (Info, error) {
	res, err := http.Get(host)
	if err != nil {
		return Info{}, errors.Wrap(ErrKuiperServer, err)

	}
	defer res.Body.Close()

	var i Info
	err = json.NewDecoder(res.Body).Decode(&i)
	if err != nil {
		return Info{}, errors.Wrap(ErrKuiperServer, err)
	}

	return i, nil
}

func (re *reService) CreateStream(ctx context.Context, token, name, topic, row string, update bool) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	_, err = re.sdk.Channel(topic, token)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	name = prepend(ui.Id, name)
	sql := sql(name, topic, row)
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return "", ErrMalformedEntity
	}

	method := "POST"
	url := fmt.Sprintf("%s/streams", host)
	if update {
		method = "PUT"
		url = fmt.Sprintf("%s/%s", url, name)
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}

	action := "Create stream"
	status := http.StatusCreated
	if update {
		action = "Update stream"
		status = http.StatusOK
	}
	result, err := result(res, action, status)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (re *reService) ListStreams(ctx context.Context, token string) ([]string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []string{}, ErrUnauthorizedAccess
	}

	var streams []string
	url := fmt.Sprintf("%s/streams", host)
	res, err := http.Get(url)
	if err != nil {
		return streams, errors.Wrap(ErrKuiperServer, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&streams)
	if err != nil {
		return streams, errors.Wrap(ErrMalformedEntity, err)
	}

	for i, value := range streams {
		streams[i] = remove(ui.Id, value)
	}

	return streams, nil
}

func (re *reService) ViewStream(ctx context.Context, token, name string) (Stream, error) {
	var stream Stream
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return stream, ErrUnauthorizedAccess
	}

	name = prepend(ui.Id, name)
	url := fmt.Sprintf("%s/streams/%s", host, name)
	res, err := http.Get(url)
	if err != nil {
		return stream, errors.Wrap(ErrKuiperServer, err)
	}
	if res.StatusCode == http.StatusNotFound {
		return stream, errors.Wrap(ErrNotFound, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&stream)
	if err != nil {
		return stream, errors.Wrap(ErrMalformedEntity, err)
	}
	stream.Name = remove(ui.Id, stream.Name)

	return stream, nil
}

func (re *reService) Delete(ctx context.Context, token, name, kind string) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	name = prepend(ui.Id, name)
	url := fmt.Sprintf("%s/%s/%s", host, kind, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}

	result, err := result(res, "Delete "+kind, http.StatusOK)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (re *reService) CreateRule(ctx context.Context, token string, rule Rule, update bool) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	_, err = re.sdk.Channel(rule.Actions[0].Mainflux.Channel, token)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	rulePrepend(ui.Id, &rule)
	body, err := json.Marshal(rule)
	if err != nil {
		return "", ErrMalformedEntity
	}

	method := "POST"
	url := fmt.Sprintf("%s/rules", host)
	if update {
		method = "PUT"
		url = fmt.Sprintf("%s/%s", url, rule.ID)
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}

	action := "Create rule"
	status := http.StatusCreated
	if update {
		action = "Update rule"
		status = http.StatusOK
	}
	result, err := result(res, action, status)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (re *reService) ListRules(ctx context.Context, token string) ([]RuleInfo, error) {
	var rules []RuleInfo

	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return rules, ErrUnauthorizedAccess
	}

	url := fmt.Sprintf("%s/rules", host)
	res, err := http.Get(url)
	if err != nil {
		return rules, errors.Wrap(ErrKuiperServer, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&rules)
	if err != nil {
		return rules, errors.Wrap(ErrMalformedEntity, err)
	}

	for i, value := range rules {
		rules[i].ID = remove(ui.Id, value.ID)
	}

	return rules, nil
}

func (re *reService) ViewRule(ctx context.Context, token, name string) (Rule, error) {
	var rule Rule
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return rule, ErrUnauthorizedAccess
	}

	name = prepend(ui.Id, name)
	url := fmt.Sprintf("%s/rules/%s", host, name)

	res, err := http.Get(url)
	if err != nil {
		return rule, errors.Wrap(ErrKuiperServer, err)
	}
	if res.StatusCode == http.StatusNotFound {
		return rule, errors.Wrap(ErrNotFound, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&rule)
	if err != nil {
		return rule, errors.Wrap(ErrMalformedEntity, err)
	}

	ruleRemove(ui.Id, &rule)

	return rule, nil
}

func (re *reService) GetRuleStatus(ctx context.Context, token, name string) (map[string]interface{}, error) {
	var status map[string]interface{}

	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return status, ErrUnauthorizedAccess
	}
	name = prepend(ui.Id, name)
	url := fmt.Sprintf("%s/rules/%s/status", host, name)

	res, err := http.Get(url)
	if err != nil {
		return status, errors.Wrap(ErrKuiperServer, err)
	}
	if res.StatusCode == http.StatusNotFound {
		return status, errors.Wrap(ErrNotFound, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&status)
	if err != nil {
		return status, errors.Wrap(ErrMalformedEntity, err)
	}

	for key, val := range status {
		delete(status, key)
		status[remove(ui.Id, key)] = val
	}

	return status, nil
}

func (re *reService) ControlRule(ctx context.Context, token, name, action string) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	name = prepend(ui.Id, name)
	url := fmt.Sprintf("%s/rules/%s/%s", host, name, action)
	res, err := http.Post(url, "", nil)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	result, err := result(res, action, http.StatusOK)
	if err != nil {
		return "", err
	}
	return result, nil
}

func sql(name, topic, row string) string {
	return fmt.Sprintf("create stream %s (%s) WITH (DATASOURCE = \"%s\" FORMAT = \"%s\" TYPE = \"%s\")", name, row, topic, FORMAT, TYPE)
}

func prefix(id string) string {
	return strings.ReplaceAll(id, "-", "") + "_"
}
func prepend(id, name string) string {
	return fmt.Sprintf("%s%s", prefix(id), name)
}

func remove(id, name string) string {
	return strings.Replace(name, prefix(id), "", 1)
}

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 //not found.
}

func rulePrepend(id string, rule *Rule) {
	rule.ID = prepend(id, rule.ID)

	// Prepend id to stream name; sql e.g. "select * from stream_name where v > 1.2;"
	words := strings.Fields(rule.SQL)
	idx := indexOf("from", words) + 1
	words[idx] = prepend(id, words[idx])
	rule.SQL = strings.Join(words, " ")
}

func ruleRemove(id string, rule *Rule) {
	rule.ID = remove(id, rule.ID)

	words := strings.Fields(rule.SQL)
	idx := indexOf("from", words) + 1
	words[idx] = remove(id, words[idx])
	rule.SQL = strings.Join(words, " ")
}

func result(res *http.Response, action string, status int) (string, error) {
	result := action + " successful."
	if res.StatusCode != status {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperServer, err)
		}
		result = action + " failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}
	return result, nil
}
