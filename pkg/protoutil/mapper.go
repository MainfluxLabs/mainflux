package protoutil

import (
	"github.com/MainfluxLabs/mainflux/pkg/proto"
)

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
