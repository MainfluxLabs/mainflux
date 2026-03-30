package protoutil

import (
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// ProtoConfigToDomain converts proto Config to domain Config.
func ProtoConfigToDomain(c *protomfx.Config) domain.ProfileConfig {
	if c == nil {
		return domain.ProfileConfig{}
	}
	tr := domain.Transformer{}
	if c.Transformer != nil {
		tr = domain.Transformer{
			DataFilters:  c.Transformer.DataFilters,
			DataField:    c.Transformer.DataField,
			TimeField:    c.Transformer.TimeField,
			TimeFormat:   c.Transformer.TimeFormat,
			TimeLocation: c.Transformer.TimeLocation,
		}
	}
	return domain.ProfileConfig{
		ContentType: c.ContentType,
		Transformer: tr,
	}
}

// PubConfigInfoToProto converts domain PubConfigInfo to proto PubConfigByKeyRes for use with messaging.FormatMessage.
func PubConfigInfoToProto(pi domain.PubConfigInfo) *protomfx.PubConfigByKeyRes {
	return &protomfx.PubConfigByKeyRes{
		PublisherID:   pi.PublisherID,
		ProfileConfig: MapToProtoConfig(pi.ProfileConfig),
	}
}

// DomainConfigToProto converts domain Config to proto Config for use with messaging.
func DomainConfigToProto(c domain.ProfileConfig) *protomfx.Config {
	tr := &protomfx.Transformer{
		DataFilters:  c.Transformer.DataFilters,
		DataField:    c.Transformer.DataField,
		TimeField:    c.Transformer.TimeField,
		TimeFormat:   c.Transformer.TimeFormat,
		TimeLocation: c.Transformer.TimeLocation,
	}
	return &protomfx.Config{
		ContentType: c.ContentType,
		Transformer: tr,
	}
}

func MapToProtoConfig(config map[string]any) *protomfx.Config {
	if config == nil {
		return &protomfx.Config{}
	}

	cfg := &protomfx.Config{}

	if v, ok := config["content_type"].(string); ok {
		cfg.ContentType = v
	}

	if t, ok := config["transformer"].(map[string]any); ok {
		tr := &protomfx.Transformer{}

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
