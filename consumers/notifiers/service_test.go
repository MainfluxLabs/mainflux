// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	ntmocks "github.com/MainfluxLabs/mainflux/consumers/notifiers/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "admin@example.com"
	groupID      = "9325aef3-5a2b-448c-bae1-5d45f86ba2aa"
	prefixID     = "fe6b4e92-cc98-425e-b0aa-"
	prefixName   = "test-notifier-"
	notifierName = "notifier-test"
	wrongValue   = "wrong-value"
	emptyValue   = ""
	svcSmtp      = "smtp-notifier"
	svcSmpp      = "smpp-notifier"
	nameKey      = "name"
	ascKey       = "asc"
	descKey      = "desc"
)

var (
	metadata      = map[string]interface{}{"test": "data"}
	validEmails   = []string{"user1@example.com", "user2@example.com"}
	validPhones   = []string{"+381610120120", "+381622220123"}
	invalidEmails = []string{"invalid@example.com", "invalid@invalid"}
	invalidPhones = []string{"0610120120", "0611111111"}
)

func newService() notifiers.Service {
	thingsC := mocks.NewThingsServiceClient(nil, nil, map[string]things.Group{token: {ID: groupID}})
	notifier := ntmocks.NewNotifier()
	notifierRepo := ntmocks.NewNotifierRepository()
	idp := uuid.NewMock()
	return notifiers.New(idp, notifier, notifierRepo, thingsC)
}

func TestConsume(t *testing.T) {
	runConsumeTest(t, svcSmtp, validEmails)
	runConsumeTest(t, svcSmpp, validPhones)
}

func runConsumeTest(t *testing.T, svcName string, validContacts []string) {
	t.Helper()
	svc := newService()
	var config, invalidConfig *protomfx.Config

	validNotifier := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	invalidNf := nf
	invalidNf.ID = "a63a8bb7-725b-4f34-89a4-857827934b1f"

	if svcName == svcSmtp {
		invalidNf.Contacts = invalidEmails
		config = &protomfx.Config{
			SmtpID: nf.ID,
		}
		invalidConfig = &protomfx.Config{
			SmtpID: invalidNf.ID,
		}
	}
	if svcName == svcSmpp {
		invalidNf.Contacts = invalidPhones
		config = &protomfx.Config{
			SmppID: nf.ID,
		}
		invalidConfig = &protomfx.Config{
			SmppID: invalidNf.ID,
		}
	}

	cases := []struct {
		desc string
		msg  protomfx.Message
		err  error
	}{
		{
			desc: "notify",
			msg:  protomfx.Message{ProfileConfig: config},
		},
		{
			desc: "notify with invalid contacts",
			msg:  protomfx.Message{ProfileConfig: invalidConfig},
			err:  notifiers.ErrNotify,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateNotifiers(t *testing.T) {
	runCreateNotifiersTest(t, svcSmtp, validEmails)
	runCreateNotifiersTest(t, svcSmpp, validPhones)
}

func runCreateNotifiersTest(t *testing.T, svcName string, validContacts []string) {
	t.Helper()
	svc := newService()
	validNf := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}

	var nfs []things.Notifier
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("%s%012d", prefixID, i+1)
		name := fmt.Sprintf("%s%012d", prefixName, i+1)
		notifier1 := validNf
		notifier1.ID = id
		notifier1.Name = name
		nfs = append(nfs, notifier1)
	}

	invalidContactsNf := validNf
	if svcName == svcSmtp {
		invalidContactsNf.Contacts = invalidEmails
	}
	if svcName == svcSmpp {
		invalidContactsNf.Contacts = invalidPhones
	}

	invalidGroupNf := validNf
	invalidGroupNf.GroupID = wrongValue

	invalidNameNf := validNf
	invalidNameNf.Name = emptyValue

	cases := []struct {
		desc      string
		notifiers []things.Notifier
		token     string
		err       error
	}{
		{
			desc:      "create new notifier",
			notifiers: nfs,
			token:     token,
			err:       nil,
		},
		{
			desc:      "create notifier with wrong credentials",
			notifiers: nfs,
			token:     wrongValue,
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "create notifier with invalid contacts",
			notifiers: []things.Notifier{invalidContactsNf},
			token:     token,
			err:       errors.ErrMalformedEntity,
		},
		{
			desc:      "create notifier with invalid group id",
			notifiers: []things.Notifier{invalidGroupNf},
			token:     token,
			err:       errors.ErrAuthorization,
		},
		{
			desc:      "create notifier with invalid name",
			notifiers: []things.Notifier{invalidNameNf},
			token:     token,
			err:       nil,
		},
	}

	for desc, tc := range cases {
		_, err := svc.CreateNotifiers(context.Background(), tc.token, tc.notifiers...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListNotifiersByGroup(t *testing.T) {
	runListNotifiersByGroupTest(t, validEmails)
	runListNotifiersByGroupTest(t, validPhones)
}

func runListNotifiersByGroupTest(t *testing.T, validContacts []string) {
	svc := newService()
	validNf := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}
	var nfs []things.Notifier
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%s%012d", prefixID, i+1)
		name := fmt.Sprintf("%s%012d", prefixName, i+1)
		notifier1 := validNf
		notifier1.ID = id
		notifier1.Name = name
		nfs = append(nfs, notifier1)
	}
	nfs, err := svc.CreateNotifiers(context.Background(), token, nfs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc         string
		token        string
		grID         string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "list the notifiers by group",
			token: token,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(nfs)),
			},
			size: uint64(len(nfs)),
			err:  nil,
		},
		{
			desc:  "list the notifiers by group with no limit",
			token: token,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Limit: 0,
			},
			size: uint64(len(nfs)),
			err:  nil,
		},
		{
			desc:  "list last notifier by group",
			token: token,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Offset: uint64(len(nfs)) - 1,
				Limit:  uint64(len(nfs)),
			},
			size: 1,
			err:  nil,
		},
		{
			desc:  "list empty set of notifiers by group",
			token: token,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Offset: uint64(len(nfs)) + 1,
				Limit:  uint64(len(nfs)),
			},
			size: 0,
			err:  nil,
		},
		{
			desc:  "list notifiers with invalid auth token",
			token: wrongValue,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:  "list notifiers with invalid group id",
			token: token,
			grID:  emptyValue,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
		{
			desc:  "list notifiers by group sorted by name ascendant",
			token: token,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(nfs)),
				Order:  nameKey,
				Dir:    ascKey,
			},
			size: uint64(len(nfs)),
			err:  nil,
		},
		{
			desc:  "list notifiers by group sorted by name descendent",
			token: token,
			grID:  groupID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(nfs)),
				Order:  nameKey,
				Dir:    descKey,
			},
			size: uint64(len(nfs)),
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListNotifiersByGroup(context.Background(), tc.token, tc.grID, tc.pageMetadata)
		size := uint64(len(page.Notifiers))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%v: expected %v got %v\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateNotifier(t *testing.T) {
	runUpdateNotifierTest(t, svcSmtp, validEmails)
	runUpdateNotifierTest(t, svcSmpp, validPhones)
}

func runUpdateNotifierTest(t *testing.T, svcName string, validContacts []string) {
	svc := newService()
	validNf := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNf)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	invalidIDNf := nf
	invalidIDNf.ID = wrongValue

	invalidContactsNf := nf
	if svcName == svcSmtp {
		invalidContactsNf.Contacts = invalidEmails
	}

	if svcName == svcSmpp {
		invalidContactsNf.Contacts = invalidPhones
	}

	invalidNameNf := nf
	invalidNameNf.Name = emptyValue

	cases := []struct {
		desc     string
		notifier things.Notifier
		token    string
		err      error
	}{
		{
			desc:     "update existing notifier",
			notifier: nf,
			token:    token,
			err:      nil,
		},
		{
			desc:     "update notifier with wrong credentials",
			notifier: nf,
			token:    emptyValue,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "update non-existing notifier",
			notifier: invalidIDNf,
			token:    token,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "create notifier with invalid contacts",
			notifier: invalidContactsNf,
			token:    token,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "create notifier with invalid name",
			notifier: invalidNameNf,
			token:    token,
			err:      nil,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateNotifier(context.Background(), tc.token, tc.notifier)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewNotifier(t *testing.T) {
	runViewNotifierTest(t, validEmails)
	runViewNotifierTest(t, validPhones)
}

func runViewNotifierTest(t *testing.T, validContacts []string) {
	t.Helper()
	svc := newService()
	validNf := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNf)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing notifier": {
			id:    nf.ID,
			token: token,
			err:   nil,
		},
		"view notifier with wrong credentials": {
			id:    nf.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		"view non-existing notifier": {
			id:    wrongValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewNotifier(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveNotifiers(t *testing.T) {
	runRemoveNotifiersTest(t, validEmails)
	runRemoveNotifiersTest(t, validPhones)
}

func runRemoveNotifiersTest(t *testing.T, validContacts []string) {
	svc := newService()
	validNf := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNf)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove notifier with wrong credentials",
			id:    nf.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing notifier",
			id:    nf.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing notifier",
			id:    wrongValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveNotifiers(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
