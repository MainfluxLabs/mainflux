// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	usersEndpoint        = "users"
	registrationEndpoint = "register"
	tokensEndpoint       = "tokens"
	passwordEndpoint     = "password"

	registerUserPath = "/auth/email-verify"
)

func (sdk mfSDK) CreateUser(u User, token string) (string, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, usersEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", usersEndpoint))
	return id, nil
}

func (sdk mfSDK) RegisterUser(u User) (string, error) {
	data, err := json.Marshal(struct {
		User User
		Path string
	}{
		u,
		registerUserPath,
	})

	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, registrationEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, "", string(CTJSON))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", usersEndpoint))
	return id, nil
}

func (sdk mfSDK) GetUser(userID, token string) (User, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, userID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return User{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return User{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return User{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var u User
	if err := json.Unmarshal(body, &u); err != nil {
		return User{}, err
	}

	return u, nil
}

func (sdk mfSDK) ListUsers(pm PageMetadata, token string) (UsersPage, error) {
	url, err := sdk.withQueryParams(sdk.usersURL, usersEndpoint, pm)
	if err != nil {
		return UsersPage{}, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return UsersPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return UsersPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UsersPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return UsersPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}
	var up UsersPage
	if err := json.Unmarshal(body, &up); err != nil {
		return UsersPage{}, err
	}

	return up, nil
}

func (sdk mfSDK) CreateToken(user User) (string, error) {
	data, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, tokensEndpoint)
	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	var tr tokenRes
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}

	return tr.Token, nil
}

func (sdk mfSDK) UpdateUser(u User, token string) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, usersEndpoint)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) UpdatePassword(oldPass, newPass, token string) error {
	ur := UserPasswordReq{
		OldPassword: oldPass,
		Password:    newPass,
	}
	data, err := json.Marshal(ur)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, passwordEndpoint)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
	}

	return nil
}
