// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

func (sdk mfSDK) SendMessage(subtopic, msg, key string) error {
	subtopic = fmt.Sprintf("/%s", strings.Replace(subtopic, ".", "/", -1))
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

func (sdk mfSDK) ReadMessages(subtopic, format, token string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/reader/messages", sdk.readerURL)
	sep := "?"
	if subtopic != "" {
		url += sep + "subtopic=" + subtopic
	}
	if format != "" {
		if subtopic != "" {
			sep = "&"
		}
		url += sep + "format=" + format
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendThingRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(ErrFailedRead, errors.New(resp.Status))
	}

	var mp map[string]interface{}
	if err := json.Unmarshal(body, &mp); err != nil {
		return nil, err
	}

	return mp, nil
}

func (sdk mfSDK) SetContentType(ct ContentType) error {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return ErrInvalidContentType
	}

	sdk.msgContentType = ct

	return nil
}
