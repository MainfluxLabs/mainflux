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
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

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
	Info() (Info, error)
	CreateStream(name, topic, row string) (string, error)
	UpdateStream(name, topic, row string) (string, error)
	ListStreams() ([]string, error)
	ViewStream(string) (Stream, error)
	DeleteStream(string) (string, error)
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

func (re *reService) CreateStream(name, topic, row string) (string, error) {
	sql := sql(name, topic, row)
	body, err := json.Marshal(map[string]string{"sql": sql})
	req, err := http.NewRequest("POST", host+"/streams", bytes.NewBuffer(body))
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

func (re *reService) UpdateStream(name, topic, row string) (string, error) {
	sql := sql(name, topic, row)
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}
	path := "/streams/" + name

	req, err := http.NewRequest("PUT", host+path, bytes.NewBuffer(body))
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

func (re *reService) ListStreams() ([]string, error) {
	var streams []string
	res, err := http.Get(host + "/streams")
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

func (re *reService) ViewStream(name string) (Stream, error) {
	var stream Stream
	res, err := http.Get(host + "/streams/" + name)
	if err != nil {
		return stream, errors.Wrap(ErrKuiperServer, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&stream)
	if err != nil {
		return stream, errors.Wrap(ErrMalformedEntity, err)
	}

	return stream, nil
}

func (re *reService) DeleteStream(name string) (string, error) {
	req, err := http.NewRequest("DELETE", host+"/streams/"+name, nil)
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
