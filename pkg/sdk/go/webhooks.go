package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const webhooksEndpoint = "webhooks"

func (sdk mfSDK) CreateWebhooks(whs []Webhook, thingID, token string) ([]Webhook, error) {
	data, err := json.Marshal(whs)
	if err != nil {
		return []Webhook{}, err
	}

	url := fmt.Sprintf("%s/things/%s/%s", sdk.webhooksURL, thingID, webhooksEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Webhook{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Webhook{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Webhook{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Webhook{}, err
	}

	var ws createWebhooksRes
	if err := json.Unmarshal(body, &ws); err != nil {
		return []Webhook{}, err
	}

	return ws.Webhooks, nil
}

func (sdk mfSDK) ListWebhooksByGroup(groupID, token string) (WebhooksPage, error) {
	url := fmt.Sprintf("%s/groups/%s/%s", sdk.webhooksURL, groupID, webhooksEndpoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return WebhooksPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return WebhooksPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WebhooksPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return WebhooksPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var wp WebhooksPage
	if err := json.Unmarshal(body, &wp); err != nil {
		return WebhooksPage{}, err
	}

	return wp, nil
}

func (sdk mfSDK) ListWebhooksByThing(thingID, token string) (WebhooksPage, error) {
	url := fmt.Sprintf("%s/things/%s/%s", sdk.webhooksURL, thingID, webhooksEndpoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return WebhooksPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return WebhooksPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WebhooksPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return WebhooksPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var wp WebhooksPage
	if err := json.Unmarshal(body, &wp); err != nil {
		return WebhooksPage{}, err
	}

	return wp, nil
}

func (sdk mfSDK) GetWebhook(webhookID, token string) (Webhook, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.webhooksURL, webhooksEndpoint, webhookID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Webhook{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Webhook{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Webhook{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Webhook{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var w Webhook
	if err := json.Unmarshal(body, &w); err != nil {
		return Webhook{}, err
	}

	return w, nil
}

func (sdk mfSDK) UpdateWebhook(wh Webhook, webhookID, token string) error {
	data, err := json.Marshal(wh)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.webhooksURL, webhooksEndpoint, webhookID)
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

func (sdk mfSDK) DeleteWebhooks(ids []string, token string) error {
	delReq := deleteWebhooksReq{WebhookIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.webhooksURL, webhooksEndpoint)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}
