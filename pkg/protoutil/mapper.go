package protoutil

import (
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
