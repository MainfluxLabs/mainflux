// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers_test

import (
	"fmt"
	"testing"

	notifiers "github.com/MainfluxLabs/mainflux/consumers/notifiers"
	ntmocks "github.com/MainfluxLabs/mainflux/consumers/notifiers/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
)

const (
	userEmail      = "user@example.com"
	otherUserEmail = "otherUser@example.com"
	invalidUser    = "invalid@example.com"
	password       = "password"
)

var (
	user      = users.User{Email: userEmail, Password: password}
	otherUser = users.User{Email: otherUserEmail, Password: password}
	usersList = []users.User{user, otherUser}
)

func newService() notifiers.Service {
	auth := mocks.NewAuthService("", usersList)
	notifier := ntmocks.NewNotifier()
	idp := uuid.NewMock()
	from := "exampleFrom"
	return notifiers.New(auth, idp, notifier, from)
}

func TestConsume(t *testing.T) {
	svc := newService()

	profile := &messaging.Profile{
		Notifier: &messaging.Notifier{
			Contacts: []string{userEmail, otherUserEmail},
		},
	}
	emptyContactProfile := &messaging.Profile{
		Notifier: &messaging.Notifier{
			Contacts: []string{},
		},
	}
	invalidContactProfile := &messaging.Profile{
		Notifier: &messaging.Notifier{
			Contacts: []string{invalidUser},
		},
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
			desc: "notify without contact",
			msg:  messaging.Message{Profile: emptyContactProfile},
			err:  notifiers.ErrNotify,
		},
		{
			desc: "notify with invalid contact",
			msg:  messaging.Message{Profile: invalidContactProfile},
			err:  notifiers.ErrNotify,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
