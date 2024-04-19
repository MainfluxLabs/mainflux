package webhooks

import (
	"encoding/json"
	"reflect"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/fxamacker/cbor/v2"
)

var ErrFormat = errors.New("unable to format payload")

type Formatter struct {
	Fields []string
}

const (
	senmlJson = "application/senml+json"
	senmlCbor = "application/senml+cbor"
)

var (
	errInvalidJson       = errors.New("invalid JSON object")
	errInvalidNestedJSON = errors.New("invalid nested JSON object")
	errInvalidCT         = errors.New("invalid content type")
)

type PayloadFormatter interface {
	FormatJSONPayload(input []byte, fieldValues []string) ([]byte, error)
	FormatSenMLPayload(payload []byte, fieldValues []string, contentType string) ([]byte, error)
}

var _ PayloadFormatter = (*Formatter)(nil)

// FormatJSONPayload method formats JSON payload based on field values
func (f Formatter) FormatJSONPayload(payload []byte, fieldValues []string) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, errors.Wrap(ErrFormat, err)
	}

	switch t := data.(type) {
	case map[string]interface{}:
		formattedPayload, err := formatSingleJson(t, fieldValues)
		if err != nil {
			return nil, errors.Wrap(ErrFormat, err)
		}

		return formattedPayload, nil
	case []interface{}:
		formattedPayload, err := formatJsonArray(t, fieldValues)
		if err != nil {
			return nil, errors.Wrap(ErrFormat, err)
		}

		return formattedPayload, nil
	default:
		return nil, errors.Wrap(ErrFormat, errInvalidJson)
	}
}

func formatSingleJson(payload map[string]interface{}, fieldValues []string) ([]byte, error) {
	formattedMap := make(map[string]interface{})

	for _, fv := range fieldValues {
		if value, ok := payload[fv]; ok {
			formattedMap[fv] = value
		}
	}

	formattedPayload, err := json.Marshal(formattedMap)
	if err != nil {
		return nil, err
	}

	return formattedPayload, nil
}

func formatJsonArray(payload []interface{}, fieldValues []string) ([]byte, error) {
	formattedMap := make(map[string]interface{})

	for _, p := range payload {
		v, ok := p.(map[string]interface{})
		if !ok {
			return nil, errInvalidNestedJSON
		}

		for _, fv := range fieldValues {
			if value, ok := v[fv]; ok {
				formattedMap[fv] = value
			}
		}
	}

	formattedPayload, err := json.Marshal(formattedMap)
	if err != nil {
		return nil, err
	}

	return formattedPayload, nil
}

// FormatSenMLPayload method formats SenML payload based on field values
func (f Formatter) FormatSenMLPayload(payload []byte, fieldValues []string, contentType string) ([]byte, error) {
	switch contentType {
	case senmlJson:
		formattedPayload, err := formatSenmlJson(payload, fieldValues)
		if err != nil {
			return nil, errors.Wrap(ErrFormat, err)
		}

		return formattedPayload, nil
	case senmlCbor:
		formattedPayload, err := formatSenmlCbor(payload, fieldValues)
		if err != nil {
			return nil, errors.Wrap(ErrFormat, err)
		}

		return formattedPayload, nil
	default:
		return nil, errors.Wrap(ErrFormat, errInvalidCT)
	}
}

func formatSenmlJson(payload []byte, fieldValues []string) ([]byte, error) {
	var senmlRecords []map[string]interface{}
	if err := json.Unmarshal(payload, &senmlRecords); err != nil {
		return nil, err
	}

	formattedMap := make(map[string]interface{})
	for _, senMLRecord := range senmlRecords {
		for _, fv := range fieldValues {
			if value, ok := senMLRecord["n"]; ok && value == fv {
				formattedMap[fv] = senMLRecord["v"]
			}
		}
	}
	formattedPayload, err := json.Marshal(formattedMap)
	if err != nil {
		return nil, err
	}

	return formattedPayload, nil
}

func formatSenmlCbor(payload []byte, fieldValues []string) ([]byte, error) {
	var senmlRecords []map[int]interface{}
	if err := cbor.Unmarshal(payload, &senmlRecords); err != nil {
		return nil, err
	}

	formattedMap := make(map[string]interface{})
	for _, senMLRecord := range senmlRecords {
		for _, fv := range fieldValues {
			if value, ok := senMLRecord[0]; ok && value == fv {
				if v, ok := senMLRecord[2]; ok && isNumber(v) {
					formattedMap[fv] = v
				}
				if v, ok := senMLRecord[3]; ok && reflect.TypeOf(v).Kind() == reflect.String {
					formattedMap[fv] = v
				}
			}
		}
	}

	formattedPayload, err := json.Marshal(formattedMap)
	if err != nil {
		return nil, err
	}

	return formattedPayload, nil
}

func isNumber(num interface{}) bool {
	switch num.(type) {
	case float32, float64, int, int64, uint, uint64:
		return true
	}

	return false
}
