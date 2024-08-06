// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/coap"
	log "github.com/MainfluxLabs/mainflux/logger"
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
	protocol     = "coap"
	authQuery    = "auth"
	startObserve = 0 // observe option value that indicates start of observation
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

	msg, err := decodeMessage(m)
	if err != nil {
		logger.Warn(fmt.Sprintf("Error decoding message: %s", err))
		resp.Code = codes.BadRequest
		sendResp(w, &resp)
		return
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
		err = handleGet(m, w.Client(), msg, key)
	case codes.POST:
		err = service.Publish(context.Background(), key, msg)
	default:
		err = errors.ErrNotFound
	}
	if err != nil {
		switch {
		case err == errBadOptions:
			resp.Code = codes.BadOption
		case err == errors.ErrNotFound:
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

func handleGet(m *mux.Message, c mux.Client, msg protomfx.Message, key string) error {
	var obs uint32
	obs, err := m.Options.Observe()
	if err != nil {
		logger.Warn(fmt.Sprintf("Error reading observe option: %s", err))
		return errBadOptions
	}
	if obs == startObserve {
		c := coap.NewClient(c, m.Token, logger)
		return service.Subscribe(context.Background(), key, msg.Channel, msg.Subtopic, c)
	}
	return service.Unsubscribe(context.Background(), key, msg.Channel, msg.Subtopic, m.Token.String())
}

func decodeMessage(msg *mux.Message) (protomfx.Message, error) {
	if msg.Options == nil {
		return protomfx.Message{}, errBadOptions
	}

	path, err := msg.Options.Path()
	if err != nil {
		return protomfx.Message{}, err
	}

	subtopic, err := messaging.ExtractSubtopic(path)
	if err != nil {
		return protomfx.Message{}, messaging.ErrMalformedSubtopic

	}

	subject, err := messaging.CreateSubject(subtopic)
	if err != nil {
		return protomfx.Message{}, err
	}

	ret := protomfx.Message{
		Protocol: protocol,
		Subtopic: subject,
		Payload:  []byte{},
		Created:  time.Now().UnixNano(),
	}

	if msg.Body != nil {
		buff, err := ioutil.ReadAll(msg.Body)
		if err != nil {
			return ret, err
		}
		ret.Payload = buff
	}
	return ret, nil
}

func parseKey(msg *mux.Message) (string, error) {
	if obs, _ := msg.Options.Observe(); obs != 0 && msg.Code == codes.GET {
		return "", nil
	}
	authKey, err := msg.Options.GetString(message.URIQuery)
	if err != nil {
		return "", err
	}
	vars := strings.Split(authKey, "=")
	if len(vars) != 2 || vars[0] != authQuery {
		return "", errors.ErrAuthorization
	}
	return vars[1], nil
}
