// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	ntmocks "github.com/MainfluxLabs/mainflux/consumers/notifiers/mocks"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token           = "admin@example.com"
	userEmail       = "user@example.com"
	otherUserEmail  = "otherUser@example.com"
	phoneNum        = "+381610120120"
	invalidPhoneNum = "0610120120"
	invalidUser     = "invalid@example.com"
	password        = "password"
	groupID         = "9325aef3-5a2b-448c-bae1-5d45f86ba2aa"
	wrongValue      = "wrong-value"
	emptyValue      = ""
)

var (
	subtopics                = []string{"subtopic1", "subtopic2"}
	user                     = users.User{Email: userEmail, Password: password}
	otherUser                = users.User{Email: otherUserEmail, Password: password}
	usersList                = []users.User{user, otherUser}
	validContacts            = []string{userEmail, phoneNum}
	invalidContacts          = []string{invalidUser, invalidPhoneNum}
	validNotifier            = things.Notifier{GroupID: groupID, Subtopics: subtopics, Contacts: validContacts}
	validNotifier2           = things.Notifier{GroupID: groupID, Subtopics: []string{subtopics[0]}, Contacts: validContacts}
	invalidContactsNotifier  = things.Notifier{GroupID: groupID, Subtopics: subtopics, Contacts: invalidContacts}
	invalidSubtopicsNotifier = things.Notifier{GroupID: groupID, Subtopics: []string{""}, Contacts: validContacts}
	invalidGroupNotifier     = things.Notifier{GroupID: emptyValue, Subtopics: subtopics, Contacts: validContacts}
)

func newService() notifiers.Service {
	auth := mocks.NewAuthService("", usersList)
	thingsC := mocks.NewThingsServiceClient(nil, map[string]string{token: groupID}, nil)
	notifier := ntmocks.NewNotifier()
	notifierRepo := ntmocks.NewNotifierRepository()
	idp := uuid.NewMock()
	from := "exampleFrom"
	return notifiers.New(auth, idp, notifier, from, notifierRepo, thingsC)
}

func TestConsume(t *testing.T) {
	svc := newService()

	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier, validNotifier2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]
	nf2 := nfs[1]

	invalidNf := nfs[0]
	invalidNf.ID = "a63a8bb7-725b-4f34-89a4-857827934b1f"
	invalidNf.Contacts = invalidContacts

	profile := &messaging.Profile{
		SmtpID: nf.ID,
		SmppID: "",
	}

	invalidContactProfile := &messaging.Profile{
		SmtpID: invalidNf.ID,
		SmppID: nf2.ID,
	}

	emptyNotifiersProfile := &messaging.Profile{
		SmtpID: "",
		SmppID: "",
	}

	cases := []struct {
		desc string
		msg  messaging.Message
		err  error
	}{
		{
			desc: "notify success",
			msg:  messaging.Message{Profile: profile},
		},
		{
			desc: "notify without notifiers",
			msg:  messaging.Message{Profile: emptyNotifiersProfile},
			err:  apiutil.ErrMissingID,
		},
		{
			desc: "notify with invalid contacts",
			msg:  messaging.Message{Profile: invalidContactProfile},
			err:  notifiers.ErrNotify,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateNotifiers(t *testing.T) {
	svc := newService()

	nfs := []things.Notifier{validNotifier, invalidContactsNotifier, invalidSubtopicsNotifier, invalidGroupNotifier}

	cases := []struct {
		desc      string
		notifiers []things.Notifier
		token     string
		err       error
	}{
		{
			desc:      "create new notifier",
			notifiers: []things.Notifier{nfs[0]},
			token:     token,
			err:       nil,
		},
		{
			desc:      "create notifier with wrong credentials",
			notifiers: []things.Notifier{nfs[0]},
			token:     wrongValue,
			err:       errors.ErrAuthorization,
		},
		{
			desc:      "create notifier with invalid contacts",
			notifiers: []things.Notifier{nfs[1]},
			token:     token,
			err:       nil,
		},
		{
			desc:      "create notifier with invalid subtopics",
			notifiers: []things.Notifier{nfs[2]},
			token:     token,
			err:       nil,
		},
		{
			desc:      "create notifier with invalid group id",
			notifiers: []things.Notifier{nfs[3]},
			token:     token,
			err:       errors.ErrAuthorization,
		},
	}

	for desc, tc := range cases {
		_, err := svc.CreateNotifiers(context.Background(), tc.token, tc.notifiers...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListNotifiersByGroup(t *testing.T) {
	svc := newService()

	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc      string
		notifiers []things.Notifier
		token     string
		grID      string
		err       error
	}{
		{
			desc:      "list the notifiers",
			notifiers: nfs,
			token:     token,
			grID:      groupID,
			err:       nil,
		},
		{
			desc:      "list notifiers with invalid auth token",
			notifiers: []things.Notifier{},
			token:     wrongValue,
			grID:      groupID,
			err:       errors.ErrAuthorization,
		},
		{
			desc:      "list notifiers with invalid group id",
			notifiers: []things.Notifier{},
			token:     token,
			err:       errors.ErrAuthorization,
		},
	}

	for desc, tc := range cases {
		whs, err := svc.ListNotifiersByGroup(context.Background(), tc.token, tc.grID)
		assert.Equal(t, tc.notifiers, whs, fmt.Sprintf("%v: expected %v got %v\n", desc, tc.notifiers, whs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateNotifier(t *testing.T) {
	svc := newService()
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	invalidIDNf := nf
	invalidIDNf.ID = emptyValue

	invalidContactsNf := nf
	invalidContactsNf.Contacts = invalidContacts

	invalidSubtopicsNf := nf
	invalidSubtopicsNf.Subtopics = []string{""}

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
			err:      errors.ErrAuthorization,
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
			err:      nil,
		},
		{
			desc:     "create notifier with invalid subtopics",
			notifier: invalidSubtopicsNf,
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
	svc := newService()
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
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
			err:   errors.ErrAuthorization,
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
	svc := newService()
	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove existing notifier",
			id:    nf.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove notifier with wrong credentials",
			id:    nf.ID,
			token: wrongValue,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveNotifiers(context.Background(), tc.token, groupID, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
