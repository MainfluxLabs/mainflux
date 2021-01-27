//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package re

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	url = "http://localhost:9081"
)

var (
	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrKuiperSever indicates internal kuiper rules engine server error
	ErrKuiperSever = errors.New("kuiper internal server error")
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
	// Ping compares a given string with secret
	Info() (Info, error)
	// CreateStream
	CreateStream(...string) (string, error)
	ListStreams() ([]string, error)
	ViewStream(string) (Stream, error)
}

type reService struct {
	secret string
}

var _ Service = (*reService)(nil)

// New instantiates the re service implementation.
func New(secret string) Service {
	return &reService{
		secret: secret,
	}
}

func (re *reService) Info() (Info, error) {
	res, err := http.Get(url)
	if err != nil {
		return Info{}, errors.Wrap(ErrKuiperSever, err)

	}
	defer res.Body.Close()

	var i Info
	err = json.NewDecoder(res.Body).Decode(&i)
	if err != nil {
		return Info{}, errors.Wrap(ErrKuiperSever, err)
	}

	return i, nil
}

func (re *reService) CreateStream(params ...string) (string, error) {
	sql := params[0]
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}

	path := "/streams"
	action := "creation"
	status := http.StatusCreated
	method := "POST"
	if len(params) > 1 {
		path += "/" + params[1]
		action = "update"
		status = http.StatusOK
		method = "PUT"
	}

	req, err := http.NewRequest(method, url+path, bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(ErrKuiperSever, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperSever, err)
	}

	result := "Steam " + action + " successful."
	if res.StatusCode != status {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperSever, err)
		}

		result = "Stream " + action + " failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}

	return result, nil
}

func (re *reService) UpdateStream(params ...string) (string, error) {
	sql := params[0]
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}

	path := "/streams"
	action := "creation"
	status := http.StatusCreated
	method := "POST"
	if len(params) > 1 {
		path += "/" + params[1]
		action = "update"
		status = http.StatusOK
		method = "PUT"
	}

	req, err := http.NewRequest(method, url+path, bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(ErrKuiperSever, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(ErrKuiperSever, err)
	}

	result := "Steam " + action + " successful."
	if res.StatusCode != status {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperSever, err)
		}

		result = "Stream " + action + " failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}

	return result, nil
}

func (re *reService) ListStreams() ([]string, error) {
	var streams []string
	res, err := http.Get(url + "/streams")
	if err != nil {
		return streams, errors.Wrap(ErrKuiperSever, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&streams)
	if err != nil {
		return streams, errors.Wrap(ErrMalformedEntity, err)
	}

	return streams, nil
}

func (re *reService) ViewStream(id string) (Stream, error) {
	var stream Stream
	res, err := http.Get(url + "/streams/" + id)
	if err != nil {
		return stream, errors.Wrap(ErrKuiperSever, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&stream)
	if err != nil {
		return stream, errors.Wrap(ErrMalformedEntity, err)
	}

	return stream, nil
}
