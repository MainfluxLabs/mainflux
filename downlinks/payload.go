// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

const (
	contentType = "Content-Type"
	jsonFormat  = "json"
	xmlFormat   = "xml"
)

// payload is used for XML-to-map unmarshalling inside formatPayload.
type payload struct {
	data map[string]any
}

// formatPayload reads the response body and returns JSON, converting XML to JSON
// or marshalling error info for unknown content types.
func formatPayload(response *http.Response) ([]byte, error) {
	defer response.Body.Close()

	resPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	format := getFormat(response.Header.Get(contentType))

	switch format {
	case xmlFormat:
		var mappedData payload
		if err := xml.Unmarshal(resPayload, &mappedData); err != nil {
			return nil, err
		}

		filteredData := removeUnderscoreKeys(mappedData.data)
		return json.Marshal(filteredData)

	case jsonFormat:
		return resPayload, nil

	default:
		errorInfo := map[string]any{
			"error":            string(resPayload),
			"http_status":      response.Status,
			"status_code":      response.StatusCode,
			"response_headers": response.Header,
			"request_method":   response.Request.Method,
			"request_url":      response.Request.URL.String(),
		}
		return json.Marshal(errorInfo)
	}
}

func (p *payload) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	p.data = make(map[string]any)
	stack := []map[string]any{p.data}
	nameStack := []xml.Name{start.Name}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			node := map[string]any{}
			parent := stack[len(stack)-1]
			currentName := elem.Name.Local

			if existing, exists := parent[currentName]; exists {
				switch v := existing.(type) {
				case []any:
					parent[currentName] = append(v, node)
				case map[string]any:
					parent[currentName] = []any{v, node}
				}
			} else {
				parent[currentName] = node
			}

			stack = append(stack, node)
			nameStack = append(nameStack, elem.Name)

		case xml.EndElement:
			stack = stack[:len(stack)-1]
			nameStack = nameStack[:len(nameStack)-1]

		case xml.CharData:
			val := strings.TrimSpace(string(elem))
			if val == "" {
				continue
			}

			current := stack[len(stack)-1]
			if len(current) == 0 {
				parent := stack[len(stack)-2]
				name := nameStack[len(nameStack)-1].Local
				parent[name] = val
			} else {
				current["#text"] = val
			}
		}
	}
}

func getFormat(ct string) string {
	switch {
	case strings.Contains(ct, jsonFormat):
		return jsonFormat
	case strings.Contains(ct, xmlFormat):
		return xmlFormat
	default:
		return ""
	}
}

func removeUnderscoreKeys(data map[string]any) map[string]any {
	filteredData := make(map[string]any)

	for key, value := range data {
		if key == "_" {
			continue
		}

		if v, ok := value.(map[string]any); ok {
			filteredData[key] = removeUnderscoreKeys(v)
			continue
		}

		filteredData[key] = value
	}

	return filteredData
}
