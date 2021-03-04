package mocks

import (
	"net/http"
	"strings"

	re "github.com/mainflux/mainflux/rules"
)

type kuiper struct {
	url     string
	streams map[string]string
	rules   map[string]re.Rule
}

var _ re.KuiperSDK = (*kuiper)(nil)

func NewKuiperSDK(url string) re.KuiperSDK {
	return &kuiper{
		url:     url,
		streams: make(map[string]string),
		rules:   make(map[string]re.Rule),
	}
}

func (k *kuiper) Info() (re.Info, error) {
	return re.Info{
		Version:       "1.00",
		Os:            "Linux",
		UpTimeSeconds: 0,
	}, nil
}

func (k *kuiper) CreateStream(sql string) (*http.Response, error) {
	var res http.Response
	res.StatusCode = http.StatusCreated
	res.Body = http.NoBody

	n := name(sql)
	if _, ok := k.streams[n]; ok {
		res.StatusCode = http.StatusConflict
		return &res, nil
	}

	k.streams[n] = sql

	return &res, nil
}

func (k *kuiper) UpdateStream(sql, name string) (*http.Response, error) {
	var res http.Response
	res.StatusCode = http.StatusNotFound
	res.Body = http.NoBody

	if _, ok := k.streams[name]; !ok {
		return &res, nil
	}

	k.streams[name] = sql
	res.StatusCode = http.StatusOK

	return &res, nil
}

func (k *kuiper) ShowStreams() ([]string, error) {
	var names []string

	for chanID := range k.streams {
		names = append(names, chanID)
	}

	return names, nil
}

func (k *kuiper) DescribeStream(name string) (*re.StreamInfo, error) {
	if _, ok := k.streams[name]; !ok {
		return &re.StreamInfo{}, re.ErrNotFound
	}

	info := re.StreamInfo{}
	info.Name = name
	fields(k.streams[name], &info)

	return &info, nil
}

func (k *kuiper) Drop(name, kind string) (*http.Response, error) {
	var res http.Response
	res.StatusCode = http.StatusBadRequest
	res.Body = http.NoBody

	if kind == "streams" {
		if _, ok := k.streams[name]; !ok {
			return &res, nil
		}
		delete(k.streams, name)
		res.StatusCode = http.StatusOK
		return &res, nil
	}
	if kind == "rules" {
		if _, ok := k.rules[name]; !ok {
			return &res, nil
		}
		delete(k.rules, name)
		res.StatusCode = http.StatusOK
		return &res, nil
	}

	return &res, nil
}

func (k *kuiper) CreateRule(rule re.Rule) (*http.Response, error) {
	var res http.Response
	res.StatusCode = http.StatusCreated
	res.Body = http.NoBody

	if _, ok := k.rules[rule.ID]; ok {
		res.StatusCode = http.StatusConflict
		return &res, nil
	}

	k.rules[rule.ID] = rule

	return &res, nil

}

func (k *kuiper) UpdateRule(rule re.Rule) (*http.Response, error) {
	var res http.Response
	res.StatusCode = http.StatusOK
	res.Body = http.NoBody

	if _, ok := k.rules[rule.ID]; !ok {
		res.StatusCode = http.StatusNotFound
		return &res, nil
	}

	k.rules[rule.ID] = rule

	return &res, nil
}

func (k *kuiper) ShowRules() ([]re.RuleInfo, error) {
	var ruleInfos []re.RuleInfo

	for _, v := range k.rules {
		ruleInfos = append(ruleInfos, re.RuleInfo{
			ID:     v.ID,
			Status: "Running",
		})
	}

	return ruleInfos, nil
}

func (k *kuiper) DescribeRule(name string) (*re.Rule, error) {
	if _, ok := k.rules[name]; !ok {
		return &re.Rule{}, re.ErrNotFound
	}

	r := k.rules[name]
	return &r, nil
}

func (k *kuiper) GetRuleStatus(name string) (map[string]interface{}, error) {
	var status map[string]interface{}
	if _, ok := k.rules[name]; !ok {
		return status, re.ErrNotFound
	}

	return status, nil
}

func (k *kuiper) ControlRule(name, action string) (*http.Response, error) {
	var res http.Response
	res.StatusCode = http.StatusOK
	res.Body = http.NoBody

	if _, ok := k.rules[name]; !ok {
		res.StatusCode = http.StatusNotFound
	}

	return &res, nil
}

func name(sql string) string {
	tokens := strings.Split(sql, " ")
	return tokens[2]
}

func fields(sql string, info *re.StreamInfo) {
	fieldStr := sql[strings.Index(sql, "(")+1 : strings.Index(sql, ")")]
	fieldArr := strings.Split(fieldStr, ", ")
	for _, v := range fieldArr {
		info.StreamFields = append(info.StreamFields, struct {
			Name      string "json:\"Name\""
			FieldType string "json:\"FieldType\""
		}{strings.Split(v, " ")[0], strings.Split(v, " ")[1]})
	}
}
