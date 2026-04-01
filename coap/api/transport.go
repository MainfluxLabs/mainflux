// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/coap"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-zoo/bone"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol            = "coap"
	authQuery           = "auth"
	authTypeQuery       = "type"
	startObserve        = 0 // observe option value that indicates start of observation
	topicPrefixThings   = "things"
	topicPrefixGroups   = "groups"
	topicSuffixCommands = "commands"
	topicSuffixMessages = "messages"
)

var errBadOptions = errors.New("bad options")

var (
	logger  log.Logger
	service coap.Service
)

// MakeHTTPHandler returns a HTTP handler for API endpoints.
func MakeHTTPHandler() http.Handler {
	b := bone.New()
	b.GetFunc("/health", mainflux.Health(protocol))
	b.Handle("/metrics", promhttp.Handler())

	return b
}

// MakeCoAPHandler creates handler for CoAP messages.
func MakeCoAPHandler(svc coap.Service, l log.Logger) mux.HandlerFunc {
	logger = l
	service = svc

	return handler
}

func sendResp(w mux.ResponseWriter, resp *message.Message) {
	if err := w.Client().WriteMessage(resp); err != nil {
		logger.Warn(fmt.Sprintf("Can't set response: %s", err))
	}
}

func handler(w mux.ResponseWriter, m *mux.Message) {
	resp := message.Message{
		Code:    codes.Content,
		Token:   m.Token,
		Context: m.Context,
		Options: make(message.Options, 0, 16),
	}

	key, err := parseKey(m)
	if err != nil {
		logger.Warn(fmt.Sprintf("Error parsing auth: %s", err))
		resp.Code = codes.Unauthorized
		sendResp(w, &resp)
		return
	}

	switch m.Code {
	case codes.GET:
		err = handleGet(m, w.Client(), key)
	case codes.POST:
		msg, decErr := decodeMessage(m)
		if decErr != nil {
			logger.Warn(fmt.Sprintf("Error decoding message: %s", decErr))
			resp.Code = codes.BadRequest
			sendResp(w, &resp)
			return
		}
		err = handlePost(m, msg, key)
	default:
		err = dbutil.ErrNotFound
	}
	if err != nil {
		switch {
		case err == errBadOptions:
			resp.Code = codes.BadOption
		case err == dbutil.ErrNotFound:
			resp.Code = codes.NotFound
		case errors.Contains(err, errors.ErrAuthorization),
			errors.Contains(err, errors.ErrAuthentication):
			resp.Code = codes.Unauthorized
		default:
			resp.Code = codes.InternalServerError
		}
		sendResp(w, &resp)
	}
}

func handlePost(m *mux.Message, msg protomfx.Message, key domain.ThingKey) error {
	path, err := m.Options.Path()
	if err != nil {
		return errBadOptions
	}

	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 4)
	if len(parts) >= 3 {
		prefix, id, suffix := parts[0], parts[1], parts[2]
		subtopicPath := ""
		if len(parts) == 4 {
			subtopicPath = parts[3]
		}
		switch suffix {
		case topicSuffixCommands:
			if msg.Subtopic, err = messaging.NormalizeSubtopic(subtopicPath); err != nil {
				return err
			}
			switch prefix {
			case topicPrefixThings:
				return service.SendCommandToThing(context.Background(), key, id, msg)
			case topicPrefixGroups:
				return service.SendCommandToGroup(context.Background(), key, id, msg)
			}
		case topicSuffixMessages:
			if msg.Subtopic, err = messaging.NormalizeSubtopic(subtopicPath); err != nil {
				return err
			}
			return service.Publish(context.Background(), key, msg)
		}
	}

	// Path is used as subtopic directly (e.g. "home/room/temperature").
	if msg.Subtopic, err = messaging.NormalizeSubtopic(path); err != nil {
		return err
	}
	return service.Publish(context.Background(), key, msg)
}

func handleGet(m *mux.Message, c mux.Client, key domain.ThingKey) error {
	var obs uint32
	obs, err := m.Options.Observe()
	if err != nil {
		logger.Warn(fmt.Sprintf("Error reading observe option: %s", err))
		return errBadOptions
	}

	path, err := m.Options.Path()
	if err != nil {
		return errBadOptions
	}
	subtopic, err := messaging.NormalizeSubtopic(strings.TrimPrefix(path, "/"))
	if err != nil {
		return err
	}

	if obs == startObserve {
		c := coap.NewClient(c, m.Token, logger)
		return service.Subscribe(context.Background(), key, subtopic, c)
	}
	return service.Unsubscribe(context.Background(), key, subtopic, m.Token.String())
}

func decodeMessage(msg *mux.Message) (protomfx.Message, error) {
	ret := protomfx.Message{
		Protocol: protocol,
		Payload:  []byte{},
		Created:  time.Now().UnixNano(),
	}

	if msg.Body != nil {
		buff, err := io.ReadAll(msg.Body)
		if err != nil {
			return ret, err
		}
		ret.Payload = buff
	}

	return ret, nil
}

func parseKey(msg *mux.Message) (domain.ThingKey, error) {
	if obs, _ := msg.Options.Observe(); obs != 0 && msg.Code == codes.GET {
		return domain.ThingKey{}, nil
	}

	queries, err := msg.Options.Queries()
	if err != nil {
		return domain.ThingKey{}, err
	}

	var thingKey domain.ThingKey

	for _, query := range queries {
		parts := strings.Split(query, "=")
		if len(parts) != 2 {
			return domain.ThingKey{}, errors.ErrAuthentication
		}

		switch parts[0] {
		case authQuery:
			thingKey.Value = parts[1]
		case authTypeQuery:
			thingKey.Type = parts[1]
		}
	}

	if err := apiutil.ValidateThingKey(thingKey); err != nil {
		return domain.ThingKey{}, err
	}

	return thingKey, nil
}
