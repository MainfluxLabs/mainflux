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
	url     = "localhost"
	token   = "token"
	email   = "angry_albattani@email.com"
	channel = "103ec2f2-2034-4d9e-8039-13f4efd36b04"
)

var (
	stream = rules.Stream{}
)

func newService(tokens map[string]string, channels map[string]string) rules.Service {
	auth := mocks.NewAuthServiceClient(tokens)
	things := mocks.NewThingsClient(channels)
	logger, err := logger.New(os.Stdout, "info")
	if err != nil {
		log.Fatalf(err.Error())
	}
	kuiper := mocks.NewKuiperSDK(url)
	return rules.New(kuiper, auth, things, logger)
}

func TestCreateThings(t *testing.T) {
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
			owner:   email,
			token:   token,
			stream:  stream,
			channel: channel,
			err:     nil,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateStream(context.Background(), tc.token, tc.stream)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
