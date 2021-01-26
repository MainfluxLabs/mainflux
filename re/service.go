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

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Ping compares a given string with secret
	Info() (Info, error)
	// CreateStream
	CreateStream(string) (string, error)
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

func (re *reService) CreateStream(sql string) (string, error) {
	body := map[string]string{"sql": sql}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}

	res, err := http.Post(url+"/streams", "application/json",
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", errors.Wrap(ErrKuiperSever, err)
	}

	result := "Successfully created stream."
	if res.StatusCode != http.StatusCreated {
		defer res.Body.Close()
		reason, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", errors.Wrap(ErrKuiperSever, err)
		}

		result = "Stream creation failed. Kuiper http status: " + strconv.Itoa(res.StatusCode) + ". " + string(reason)
	}

	return result, nil
}
