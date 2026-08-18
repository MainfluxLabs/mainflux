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
	refreshEndpoint      = "refresh"
	passwordEndpoint     = "password"

	redirectPathEmailVerify = "/auth/email-verify"
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
		User         User
		RedirectPath string `json:"redirect_path"`
	}{
		u,
		redirectPathEmailVerify,
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

func (sdk mfSDK) CreateToken(user User) (TokenPair, error) {
	data, err := json.Marshal(user)
	if err != nil {
		return TokenPair{}, err
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, tokensEndpoint)
	tokens, err := sdk.postTokens(url, data, http.StatusCreated, ErrFailedCreation)
	if err != nil {
		return TokenPair{}, err
	}

	if sdk.autoRefresh {
		sdk.session.set(tokens)
	}

	return tokens, nil
}

// RefreshToken exchanges a refresh token for a new access token. The refresh
// token is returned unchanged: it is stateless server-side and its expiry is
// not extended, so a session ends at the refresh token's original lifetime.
func (sdk mfSDK) RefreshToken(refreshToken string) (TokenPair, error) {
	data, err := json.Marshal(refreshReq{RefreshToken: refreshToken})
	if err != nil {
		return TokenPair{}, err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, tokensEndpoint, refreshEndpoint)

	return sdk.postTokens(url, data, http.StatusOK, ErrFailedFetch)
}

func (sdk mfSDK) postTokens(url string, data []byte, wantStatus int, failure error) (TokenPair, error) {
	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return TokenPair{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenPair{}, err
	}

	if resp.StatusCode != wantStatus {
		return TokenPair{}, errors.Wrap(failure, errors.New(resp.Status))
	}

	var tr tokenRes
	if err := json.Unmarshal(body, &tr); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  tr.Token,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    tr.ExpiresAt,
	}, nil
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
