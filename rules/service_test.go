package rules_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/rules"
	"github.com/mainflux/mainflux/rules/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	url      = "localhost"
	token    = "token"
	token2   = "token2"
	wrong    = "wrong"
	email    = "angry_albattani@email.com"
	email2   = "xenodochial_goldwasser@email.com"
	channel  = "103ec2f2-2034-4d9e-8039-13f4efd36b04"
	channel2 = "243fec72-7cf7-4bca-ac87-44a53b318510"
	sql      = "select * from stream where v > 1.2;"
)

var (
	stream = rules.Stream{
		Topic: channel,
	}
	stream2 = rules.Stream{
		Topic: channel2,
	}
	rule  = createRule("rule", channel)
	rule2 = createRule("rule2", channel2)
)

func newService(users map[string]string, channels map[string]string) rules.Service {
	// map[token]email
	auth := mocks.NewAuthServiceClient(users)
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
		desc   string
		token  string
		stream rules.Stream
		err    error
	}{
		{
			desc:   "create non-existing stream when user owns channel",
			token:  token,
			stream: stream,
			err:    nil,
		},
		{
			desc:  "wrong token",
			token: wrong,
			err:   rules.ErrUnauthorizedAccess,
		},
		{
			desc:   "create existing stream when user owns channel",
			token:  token,
			stream: stream,
			err:    rules.ErrKuiperServer,
		},
		{
			desc:   "create non-existing stream when user does not own channel",
			token:  token,
			stream: stream2,
			err:    rules.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := svc.CreateStream(context.Background(), tc.token, tc.stream)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateStream(t *testing.T) {
	svc := newService(map[string]string{token: email}, map[string]string{channel: email})

	_, err := svc.CreateStream(context.Background(), token, stream)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		stream rules.Stream
		err    error
	}{
		{
			desc:   "update non-existing stream when user owns channel",
			token:  token,
			stream: stream2,
			err:    rules.ErrNotFound,
		},
		{
			desc:  "wrong token",
			token: wrong,
			err:   rules.ErrUnauthorizedAccess,
		},
		{
			desc:   "update existing stream when user owns channel",
			token:  token,
			stream: stream,
			err:    nil,
		},
		{
			desc:   "update non-existing stream when user does not own channel",
			token:  token,
			stream: stream2,
			err:    rules.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := svc.UpdateStream(context.Background(), tc.token, tc.stream)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListStreams(t *testing.T) {
	numChans := 10
	channels := make(map[string]string)
	for i := 0; i < numChans; i++ {
		channels[strconv.Itoa(i)] = email
	}
	mult := 3
	for i := numChans; i < numChans*mult; i++ {
		channels[strconv.Itoa(i)] = email2
	}

	svc := newService(map[string]string{token: email, token2: email2}, channels)
	for i := 0; i < numChans; i++ {
		id := strconv.Itoa(i)
		_, err := svc.CreateStream(context.Background(), token, rules.Stream{
			Name:  id,
			Topic: id,
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}
	for i := numChans; i < numChans*mult; i++ {
		id := strconv.Itoa(i)
		_, err := svc.CreateStream(context.Background(), token2, rules.Stream{
			Name:  id,
			Topic: id,
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		token    string
		numChans int
		err      error
	}{
		{
			desc:     "1st correct token",
			token:    token,
			numChans: numChans,
			err:      nil,
		},
		{
			desc:     "wrong token",
			token:    wrong,
			numChans: 0,
			err:      rules.ErrUnauthorizedAccess,
		},
		{
			desc:     "2nd correct token",
			token:    token2,
			numChans: numChans * (mult - 1),
			err:      nil,
		},
	}
	for _, tc := range cases {
		chans, err := svc.ListStreams(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.numChans, len(chans), fmt.Sprintf("%s: expected %d got %d streams\n", tc.desc, tc.numChans, len(chans)))
	}
}

func TestDeleteStreams(t *testing.T) {
	ctx := context.Background()

	users := map[string]string{token: email, token2: email2}
	numChans := 10
	channels := make(map[string]string)
	for i := 0; i < numChans; i++ {
		channels[strconv.Itoa(i)] = email
	}
	for i := numChans; i < numChans*2; i++ {
		channels[strconv.Itoa(i)] = email2
	}
	svc := newService(users, channels)

	for i := 0; i < numChans; i++ {
		id := strconv.Itoa(i)
		_, err := svc.CreateStream(ctx, token, rules.Stream{
			Name:  id,
			Topic: id,
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}
	for i := 0; i < numChans; i++ {
		_, err := svc.CreateStream(ctx, token2, rules.Stream{
			Name: strconv.Itoa(i),
			// i + numChans - chan IDs must be unique
			Topic: strconv.Itoa(i + numChans),
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc      string
		token     string
		numDel    int
		numRemain int
		err       error
	}{
		{
			desc:      "1st correct token",
			token:     token,
			numDel:    numChans / 2,
			numRemain: numChans / 2,
			err:       nil,
		},
		{
			desc:      "wrong token",
			token:     wrong,
			numDel:    1,
			numRemain: 0,
			err:       rules.ErrUnauthorizedAccess,
		},
		{
			desc:      "2nd correct token",
			token:     token2,
			numDel:    numChans,
			numRemain: 0,
			err:       nil,
		},
	}

	for _, tc := range cases {
		for i := 0; i < tc.numDel; i++ {
			_, err := svc.Delete(ctx, tc.token, strconv.Itoa(i), "streams")
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
		entities, err := svc.ListStreams(ctx, tc.token)
		// if token != wrong
		if !errors.Contains(err, rules.ErrUnauthorizedAccess) {
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		}
		assert.Equal(t, tc.numRemain, len(entities), fmt.Sprintf("%s: expected %d got %d streams\n", tc.desc, tc.numRemain, len(entities)))
	}
}

func TestCreateRule(t *testing.T) {
	svc := newService(map[string]string{token: email}, map[string]string{channel: email})

	cases := []struct {
		desc  string
		token string
		rule  rules.Rule
		err   error
	}{
		{
			desc:  "create non-existing rule when user owns channel",
			token: token,
			rule:  rule,
			err:   nil,
		},
		{
			desc:  "wrong token",
			token: wrong,
			err:   rules.ErrUnauthorizedAccess,
		},
		{
			desc:  "create existing rule when user owns channel",
			token: token,
			rule:  rule,
			err:   rules.ErrKuiperServer,
		},
		{
			desc:  "create non-existing rule when user does not own channel",
			token: token,
			rule:  rule2,
			err:   rules.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := svc.CreateRule(context.Background(), tc.token, tc.rule)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateRule(t *testing.T) {
	svc := newService(map[string]string{token: email}, map[string]string{channel: email})

	_, err := svc.CreateStream(context.Background(), token, stream)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = svc.CreateRule(context.Background(), token, rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		token string
		rule  rules.Rule
		err   error
	}{
		{
			desc:  "update non-existing rule when user owns channel",
			token: token,
			rule:  rule2,
			err:   rules.ErrNotFound,
		},
		{
			desc:  "wrong token",
			token: wrong,
			err:   rules.ErrUnauthorizedAccess,
		},
		{
			desc:  "update existing rule when user owns channel",
			token: token,
			rule:  rule,
			err:   nil,
		},
		{
			desc:  "update non-existing rule when user does not own channel",
			token: token,
			rule:  rule2,
			err:   rules.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.UpdateRule(context.Background(), tc.token, tc.rule)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListRules(t *testing.T) {
	numChans := 10
	channels := make(map[string]string)
	for i := 0; i < numChans; i++ {
		channels[strconv.Itoa(i)] = email
	}
	mult := 3
	for i := numChans; i < numChans*mult; i++ {
		channels[strconv.Itoa(i)] = email2
	}

	svc := newService(map[string]string{token: email, token2: email2}, channels)
	for i := 0; i < numChans; i++ {
		id := strconv.Itoa(i)
		_, err := svc.CreateRule(context.Background(), token, createRule(id, id))
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}
	for i := numChans; i < numChans*mult; i++ {
		id := strconv.Itoa(i)
		_, err := svc.CreateRule(context.Background(), token2, createRule(id, id))
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		token    string
		numChans int
		err      error
	}{
		{
			desc:     "1st correct token",
			token:    token,
			numChans: numChans,
			err:      nil,
		},
		{
			desc:     "wrong token",
			token:    wrong,
			numChans: 0,
			err:      rules.ErrUnauthorizedAccess,
		},
		{
			desc:     "2nd correct token",
			token:    token2,
			numChans: numChans * (mult - 1),
			err:      nil,
		},
	}
	for _, tc := range cases {
		chans, err := svc.ListRules(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.numChans, len(chans), fmt.Sprintf("%s: expected %d got %d rules\n", tc.desc, tc.numChans, len(chans)))
	}
}

func TestDeleteRules(t *testing.T) {
	ctx := context.Background()

	users := map[string]string{token: email, token2: email2}
	numChans := 10
	channels := make(map[string]string)
	for i := 0; i < numChans; i++ {
		channels[strconv.Itoa(i)] = email
	}
	for i := numChans; i < numChans*2; i++ {
		channels[strconv.Itoa(i)] = email2
	}
	svc := newService(users, channels)

	for i := 0; i < numChans; i++ {
		id := strconv.Itoa(i)
		_, err := svc.CreateRule(ctx, token, createRule(id, id))
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}
	for i := 0; i < numChans; i++ {
		id := strconv.Itoa(i)
		ch := strconv.Itoa(i + numChans)
		_, err := svc.CreateRule(ctx, token2, createRule(id, ch))
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc      string
		token     string
		numDel    int
		numRemain int
		err       error
	}{
		{
			desc:      "1st correct token",
			token:     token,
			numDel:    numChans / 2,
			numRemain: numChans / 2,
			err:       nil,
		},
		{
			desc:      "wrong token",
			token:     wrong,
			numDel:    1,
			numRemain: 0,
			err:       rules.ErrUnauthorizedAccess,
		},
		{
			desc:      "2nd correct token",
			token:     token2,
			numDel:    numChans,
			numRemain: 0,
			err:       nil,
		},
	}

	for _, tc := range cases {
		for i := 0; i < tc.numDel; i++ {
			_, err := svc.Delete(ctx, tc.token, strconv.Itoa(i), "rules")
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
		entities, err := svc.ListRules(ctx, tc.token)
		// if token != wrong
		if !errors.Contains(err, rules.ErrUnauthorizedAccess) {
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		}
		assert.Equal(t, tc.numRemain, len(entities), fmt.Sprintf("%s: expected %d got %d streams\n", tc.desc, tc.numRemain, len(entities)))
	}
}

func createRule(id, channel string) rules.Rule {
	var rule rules.Rule

	rule.ID = id
	rule.SQL = sql
	rule.Actions = append(rule.Actions, struct{ Mainflux rules.Action }{
		Mainflux: rules.Action{
			Channel: channel,
		},
	})

	return rule
}
