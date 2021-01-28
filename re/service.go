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

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	host   = "http://localhost:9081"
	FORMAT = "json"
	TYPE   = "mainflux"
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

type Stream struct {
	Name         string `json:"Name"`
	StreamFields []struct {
		Name      string `json:"Name"`
		FieldType string `json:"FieldType"`
	} `json:"StreamFields"`
	Options struct {
		DATASOURCE string `json:"DATASOURCE"`
		FORMAT     string `json:"FORMAT"`
	} `json:"Options"`
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	Info(ctx context.Context) (Info, error)
	CreateStream(ctx context.Context, token, name, topic, row string) (string, error)
	UpdateStream(ctx context.Context, token, name, topic, row string) (string, error)
	ListStreams(ctx context.Context, token string) ([]string, error)
	ViewStream(ctx context.Context, token, name string) (Stream, error)
	DeleteStream(ctx context.Context, token, name string) (string, error)
}

type reService struct {
	auth   mainflux.AuthServiceClient
	logger logger.Logger
}

var _ Service = (*reService)(nil)

// New instantiates the re service implementation.
func New(auth mainflux.AuthServiceClient, logger logger.Logger) Service {
	return &reService{
		auth:   auth,
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

func (re *reService) CreateStream(ctx context.Context, token, name, topic, row string) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	_ = ui

	// id, err = re.idProvider.ID()
	// if err != nil {
	// 	return "", err
	// }

	// owner := ui.GetEmail()

	sql := sql(name, topic, row)
	body, err := json.Marshal(map[string]string{"sql": sql})
	url := fmt.Sprintf("%s/%s", host, "streams")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}

	result := "Steam creation successful."
	if res.StatusCode != http.StatusCreated {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperServer, err)
		}
		result = "Stream creation failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}

	return result, nil
}

func (re *reService) UpdateStream(ctx context.Context, token, name, topic, row string) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	_ = ui

	sql := sql(name, topic, row)
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}
	url := fmt.Sprintf("%s/%s/%s", host, "streams", name)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}

	result := "Stream update successful."
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperServer, err)
		}

		result = "Stream update failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}

	return result, nil
}

func (re *reService) ListStreams(ctx context.Context, token string) ([]string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []string{}, ErrUnauthorizedAccess
	}
	_ = ui

	var streams []string
	url := fmt.Sprintf("%s/%s", host, "streams")
	res, err := http.Get(url)
	if err != nil {
		return streams, errors.Wrap(ErrKuiperServer, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&streams)
	if err != nil {
		return streams, errors.Wrap(ErrMalformedEntity, err)
	}

	return streams, nil
}

func (re *reService) ViewStream(ctx context.Context, token, name string) (Stream, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Stream{}, ErrUnauthorizedAccess
	}
	_ = ui

	var stream Stream
	url := fmt.Sprintf("%s/%s/%s", host, "streams", name)

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

	return stream, nil
}

func (re *reService) DeleteStream(ctx context.Context, token, name string) (string, error) {
	ui, err := re.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	_ = ui

	url := fmt.Sprintf("%s/%s/%s", host, "streams", name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperServer, err)
	}

	result := "Stream delete successful."
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperServer, err)
		}

		result = "Stream update failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}
	return result, nil
}

func sql(name, topic, row string) string {
	return fmt.Sprintf("create stream %s (%s) WITH (DATASOURCE = \"%s\" FORMAT = \"%s\" TYPE = \"%s\")", name, row, topic, FORMAT, TYPE)
}
