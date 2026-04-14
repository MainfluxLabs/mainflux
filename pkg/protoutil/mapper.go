package protoutil

import (
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// ProtoConfigToDomain converts proto Config to domain Config.
func ProtoConfigToDomain(c *protomfx.Config) *domain.ProfileConfig {
	if c == nil {
		return nil
	}

	tr := domain.Transformer{}
	if c.Transformer != nil {
		tr = domain.Transformer{
			DataFilters:  c.Transformer.GetDataFilters(),
			DataField:    c.Transformer.GetDataField(),
			TimeField:    c.Transformer.GetTimeField(),
			TimeFormat:   c.Transformer.GetTimeFormat(),
			TimeLocation: c.Transformer.GetTimeLocation(),
		}
	}

	return &domain.ProfileConfig{
		ContentType: c.GetContentType(),
		Transformer: tr,
	}
}

// DomainConfigToProto converts domain Config to proto Config for use with messaging.
func DomainConfigToProto(c *domain.ProfileConfig) *protomfx.Config {
	if c == nil {
		return nil
	}

	cfg := &protomfx.Config{
		ContentType: c.ContentType,
		Transformer: &protomfx.Transformer{
			DataFilters:  c.Transformer.DataFilters,
			DataField:    c.Transformer.DataField,
			TimeField:    c.Transformer.TimeField,
			TimeFormat:   c.Transformer.TimeFormat,
			TimeLocation: c.Transformer.TimeLocation,
		},
	}

	return cfg
}

// MapToDomainConfig converts a map to domain ProfileConfig.
func MapToDomainConfig(config map[string]any) *domain.ProfileConfig {
	if config == nil {
		return nil
	}

	cfg := &domain.ProfileConfig{}

	if v, ok := config["content_type"].(string); ok {
		cfg.ContentType = v
	}

	if t, ok := config["transformer"].(map[string]any); ok {
		tr := domain.Transformer{}

		if filters, ok := t["data_filters"].([]string); ok {
			tr.DataFilters = filters
		}
		if df, ok := t["data_field"].(string); ok {
			tr.DataField = df
		}
		if tf, ok := t["time_field"].(string); ok {
			tr.TimeField = tf
		}
		if tf, ok := t["time_format"].(string); ok {
			tr.TimeFormat = tf
		}
		if tl, ok := t["time_location"].(string); ok {
			tr.TimeLocation = tl
		}

		cfg.Transformer = tr
	}

	return cfg
}

// Map a map[string]any representing a JSON message to a *protomfx.Message.
func JSONMapMessageToProto(msg map[string]any) *protomfx.Message {
	if len(msg) == 0 {
		return nil
	}

	created, ok := msg["created"].(int64)
	if !ok {
		return nil
	}

	publisher, ok := msg["publisher"].(string)
	if !ok {
		return nil
	}

	protocol, ok := msg["protocol"].(string)
	if !ok {
		return nil
	}

	payloadMap, ok := msg["payload"].(map[string]any)
	if !ok {
		return nil
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return nil
	}

	subtopic, ok := msg["subtopic"].(string)
	if !ok {
		return nil
	}

	return &protomfx.Message{
		Publisher: publisher,
		Subtopic:  subtopic,
		Payload:   payload,
		Protocol:  protocol,
		Created:   created,
	}
}

// Map a *protomfx.Message representing a JSON message to a map[string]any with a decoded payload.
func ProtoJSONMessageToMap(msg *protomfx.Message) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}

	return map[string]any{
		"publisher": msg.Publisher,
		"subtopic":  msg.Subtopic,
		"protocol":  msg.Protocol,
		"created":   msg.Created,
		"payload":   payload,
	}, nil
}
