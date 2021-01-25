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

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Ping compares a given string with secret
	Ping(string) (string, error)
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

type info struct {
	Version       string `json:"version"`
	Os            string `json:"os"`
	UpTimeSeconds int    `json:"upTimeSeconds"`
}

func (re *reService) Ping(secret string) (string, error) {
	res, err := http.Get("http://localhost:9081")
	if err != nil {
		fmt.Printf("%+v\n", err) // output for debug

	}
	defer res.Body.Close()

	var ki info
	err = json.NewDecoder(res.Body).Decode(&ki)
	if err != nil {
		fmt.Printf("%+v\n", err) // output for debug
	}

	return fmt.Sprintf("%+v\n", ki), nil
}

func (re *reService) CreateStream(sql string) (string, error) {
	body := map[string]string{"sql": sql}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}

	res, err := http.Post("http://localhost:9081/streams", "application/json",
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
