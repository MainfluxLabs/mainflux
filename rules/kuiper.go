package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

type KuiperSDK interface {
	Info() (Info, error)
	CreateStream(sql string) (*http.Response, error)
	UpdateStream(sql, stream string) (*http.Response, error)
	ShowStreams() ([]string, error)
	DescribeStream(name string) (*StreamInfo, error)
	Drop(name, kind string) (*http.Response, error)

	CreateRule(rule Rule) (*http.Response, error)
	UpdateRule(rule Rule) (*http.Response, error)
	ShowRules() ([]RuleInfo, error)
	DescribeRule(name string) (*Rule, error)
	GetRuleStatus(name string) (map[string]interface{}, error)
	ControlRule(name, action string) (*http.Response, error)
}

type kuiper struct {
	url string
}

var _ KuiperSDK = (*kuiper)(nil)

func NewKuiperSDK(url string) KuiperSDK {
	return &kuiper{
		url: url,
	}
}

func (k *kuiper) Info() (Info, error) {
	var i Info

	res, err := http.Get(k.url)
	if err != nil {
		return i, errors.Wrap(ErrKuiperServer, err)

	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&i)
	if err != nil {
		return i, errors.Wrap(ErrKuiperServer, err)
	}

	return i, nil
}

func (k *kuiper) CreateStream(sql string) (*http.Response, error) {
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return nil, ErrMalformedEntity
	}

	url := fmt.Sprintf("%s/streams", k.url)
	res, err := http.Post(url, contentType, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}

	return res, nil
}

func (k *kuiper) UpdateStream(sql, name string) (*http.Response, error) {
	body, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return nil, ErrMalformedEntity
	}

	url := fmt.Sprintf("%s/streams/%s", k.url, name)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, ErrMalformedEntity
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}

	return res, nil
}

func (k *kuiper) ShowStreams() ([]string, error) {
	var streams []string

	url := fmt.Sprintf("%s/streams", k.url)
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

func (k *kuiper) DescribeStream(name string) (*StreamInfo, error) {
	url := fmt.Sprintf("%s/streams/%s", k.url, name)
	res, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, errors.Wrap(ErrNotFound, err)
	}
	defer res.Body.Close()

	var stream StreamInfo
	err = json.NewDecoder(res.Body).Decode(&stream)
	if err != nil {
		return nil, errors.Wrap(ErrMalformedEntity, err)
	}

	return &stream, nil
}

func (k *kuiper) Drop(name, kind string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", k.url, kind, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	return res, nil
}

func (k *kuiper) CreateRule(rule Rule) (*http.Response, error) {
	body, err := json.Marshal(rule)
	if err != nil {
		return nil, ErrMalformedEntity
	}

	url := fmt.Sprintf("%s/rules", k.url)
	res, err := http.Post(url, contentType, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	return res, nil
}

func (k *kuiper) UpdateRule(rule Rule) (*http.Response, error) {
	body, err := json.Marshal(rule)
	if err != nil {
		return nil, ErrMalformedEntity
	}

	url := fmt.Sprintf("%s/rules/%s", k.url, rule.ID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}

	return res, nil
}

func (k *kuiper) ShowRules() ([]RuleInfo, error) {
	var ruleInfos []RuleInfo

	url := fmt.Sprintf("%s/rules", k.url)
	res, err := http.Get(url)
	if err != nil {
		return ruleInfos, errors.Wrap(ErrKuiperServer, err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&ruleInfos)
	if err != nil {
		return ruleInfos, errors.Wrap(ErrMalformedEntity, err)
	}

	return ruleInfos, nil
}

func (k *kuiper) DescribeRule(name string) (*Rule, error) {
	url := fmt.Sprintf("%s/rules/%s", k.url, name)

	res, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, errors.Wrap(ErrNotFound, err)
	}
	defer res.Body.Close()

	var rule Rule
	err = json.NewDecoder(res.Body).Decode(&rule)
	if err != nil {
		return nil, errors.Wrap(ErrMalformedEntity, err)
	}

	return &rule, nil
}

func (k *kuiper) GetRuleStatus(name string) (map[string]interface{}, error) {
	var status map[string]interface{}

	url := fmt.Sprintf("%s/rules/%s/status", k.url, name)

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

	return status, nil
}

func (k *kuiper) ControlRule(name, action string) (*http.Response, error) {
	url := fmt.Sprintf("%s/rules/%s/%s", k.url, name, action)
	res, err := http.Post(url, "", nil)
	if err != nil {
		return nil, errors.Wrap(ErrKuiperServer, err)
	}
	return res, nil
}
