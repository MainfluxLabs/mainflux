package influxdb

import "github.com/MainfluxLabs/mainflux/pkg/transformers/senml"

type tags map[string]string

func senmlTags(msg senml.Message) tags {
	return tags{
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"name":      msg.Name,
	}
}
