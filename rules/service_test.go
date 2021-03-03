package rules_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/rules"
	"github.com/mainflux/mainflux/rules/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	url      = "localhost"
	token    = "token"
	wrong    = "wrong"
	email    = "angry_albattani@email.com"
	channel  = "103ec2f2-2034-4d9e-8039-13f4efd36b04"
	channel2 = "243fec72-7cf7-4bca-ac87-44a53b318510"
)

var (
	stream = rules.Stream{
		Topic: channel,
	}
	stream2 = rules.Stream{
		Topic: channel2,
	}
)

func newService(tokens map[string]string, channels map[string]string) rules.Service {
	// map[token]email
	auth := mocks.NewAuthServiceClient(tokens)
	// map[chanID]email
	things := mocks.NewThingsClient(channels)
	logger, err := logger.New(os.Stdout, "info")
	if err != nil {
		log.Fatalf(err.Error())
	}
	kuiper := mocks.NewKuiperSDK(url)
	return rules.New(kuiper, auth, things, logger)
}

func TestCreateStream(t *testing.T) {
	svc := newService(map[string]string{token: email}, map[string]string{channel: email})

	cases := []struct {
		desc    string
		owner   string
		token   string
		channel string
		stream  rules.Stream
		err     error
	}{
		{
			desc:    "create non-existing stream when user owns channel",
			owner:   email,
			token:   token,
			stream:  stream,
			channel: channel,
			err:     nil,
		},
		{
			desc:  "wrong token",
			token: wrong,
			err:   rules.ErrUnauthorizedAccess,
		},
		{
			desc:    "create existing stream when user owns channel",
			owner:   email,
			token:   token,
			stream:  stream,
			channel: channel,
			err:     rules.ErrKuiperServer,
		},
		{
			desc:    "create non-existing stream when user does not own channel",
			owner:   email,
			token:   token,
			stream:  stream2,
			channel: channel,
			err:     rules.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateStream(context.Background(), tc.token, tc.stream)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateStream(t *testing.T) {
	svc := newService(map[string]string{token: email}, map[string]string{channel: email})

	cases := []struct {
		desc    string
		owner   string
		token   string
		channel string
		stream  rules.Stream
		err     error
	}{
		{
			desc:    "update non-existing stream when user owns channel",
			owner:   email,
			token:   token,
			stream:  stream,
			channel: channel,
			err:     nil,
		},
		{
			desc:  "wrong token",
			token: wrong,
			err:   rules.ErrUnauthorizedAccess,
		},
		{
			desc:    "update existing stream when user owns channel",
			owner:   email,
			token:   token,
			stream:  stream,
			channel: channel,
			err:     nil,
		},
		{
			desc:    "update non-existing stream when user does not own channel",
			owner:   email,
			token:   token,
			stream:  stream2,
			channel: channel,
			err:     rules.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.UpdateStream(context.Background(), tc.token, tc.stream)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
