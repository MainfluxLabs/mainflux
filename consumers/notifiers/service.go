// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"
	"regexp"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/internal/email"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service represents a notification service.
type Service interface {
	// CreateNotifiers creates notifiers for certain group identified by the provided ID
	CreateNotifiers(ctx context.Context, token string, notifiers ...things.Notifier) ([]things.Notifier, error)

	// ListNotifiersByGroup retrieves data about a subset of notifiers
	// related to a certain group identified by the provided ID.
	ListNotifiersByGroup(ctx context.Context, token string, groupID string, pm things.PageMetadata) (things.NotifiersPage, error)

	// ViewNotifier retrieves data about the notifier identified with the provided ID
	ViewNotifier(ctx context.Context, token, id string) (things.Notifier, error)

	// UpdateNotifier updates the notifier identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateNotifier(ctx context.Context, token string, notifier things.Notifier) error

	// RemoveNotifiers removes the notifiers identified with the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveNotifiers(ctx context.Context, token, groupID string, id ...string) error

	consumers.Consumer
}

var _ Service = (*notifierService)(nil)

type notifierService struct {
	idp          uuid.IDProvider
	notifier     Notifier
	from         string
	svcName      string
	notifierRepo NotifierRepository
	things       protomfx.ThingsServiceClient
}

// New instantiates the subscriptions service implementation.
func New(idp uuid.IDProvider, notifier Notifier, from, svcName string, notifierRepo NotifierRepository, things protomfx.ThingsServiceClient) Service {
	return &notifierService{
		idp:          idp,
		notifier:     notifier,
		from:         from,
		svcName:      svcName,
		notifierRepo: notifierRepo,
		things:       things,
	}
}

func (ns *notifierService) Consume(message interface{}) error {
	ctx := context.Background()

	msg, ok := message.(protomfx.Message)
	if !ok {
		return errors.ErrMessage
	}

	if msg.Profile.SmtpID != "" {
		smtp, err := ns.notifierRepo.RetrieveByID(ctx, msg.Profile.SmtpID)
		err = ns.notifier.Notify(ns.from, smtp.Contacts, msg)
		if err != nil {
			return errors.Wrap(ErrNotify, err)
		}
	}

	if msg.Profile.SmppID != "" {
		smpp, err := ns.notifierRepo.RetrieveByID(ctx, msg.Profile.SmppID)
		err = ns.notifier.Notify(ns.from, smpp.Contacts, msg)
		if err != nil {
			return errors.Wrap(ErrNotify, err)
		}
	}

	return nil
}

func (ns *notifierService) CreateNotifiers(ctx context.Context, token string, notifiers ...things.Notifier) ([]things.Notifier, error) {
	nfs := []things.Notifier{}
	for _, notifier := range notifiers {
		if err := ns.validateContacts(notifier.Contacts); err != nil {
			return []things.Notifier{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		nf, err := ns.createNotifier(ctx, &notifier, token)
		if err != nil {
			return []things.Notifier{}, err
		}
		nfs = append(nfs, nf)
	}

	return nfs, nil
}

func (ns *notifierService) createNotifier(ctx context.Context, notifier *things.Notifier, token string) (things.Notifier, error) {
	_, err := ns.things.Authorize(ctx, &protomfx.AuthorizeReq{Token: token, Object: notifier.GroupID, Subject: things.GroupSub, Action: things.Editor})
	if err != nil {
		return things.Notifier{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	id, err := ns.idp.ID()
	if err != nil {
		return things.Notifier{}, err
	}
	notifier.ID = id

	nfs, err := ns.notifierRepo.Save(ctx, *notifier)
	if err != nil {
		return things.Notifier{}, err
	}

	if len(nfs) == 0 {
		return things.Notifier{}, errors.ErrCreateEntity
	}

	return nfs[0], nil
}

func (ns *notifierService) ListNotifiersByGroup(ctx context.Context, token string, groupID string, pm things.PageMetadata) (things.NotifiersPage, error) {
	_, err := ns.things.Authorize(ctx, &protomfx.AuthorizeReq{Token: token, Object: groupID, Subject: things.GroupSub, Action: things.Viewer})
	if err != nil {
		return things.NotifiersPage{}, err
	}

	notifiers, err := ns.notifierRepo.RetrieveByGroupID(ctx, groupID, pm)
	if err != nil {
		return things.NotifiersPage{}, err
	}

	return notifiers, nil
}

func (ns *notifierService) ViewNotifier(ctx context.Context, token, id string) (things.Notifier, error) {
	notifier, err := ns.notifierRepo.RetrieveByID(ctx, id)
	if err != nil {
		return things.Notifier{}, err
	}

	if _, err := ns.things.Authorize(ctx, &protomfx.AuthorizeReq{Token: token, Object: notifier.GroupID, Subject: things.GroupSub, Action: things.Viewer}); err != nil {
		return things.Notifier{}, err
	}

	return notifier, nil
}

func (ns *notifierService) UpdateNotifier(ctx context.Context, token string, notifier things.Notifier) error {
	nf, err := ns.notifierRepo.RetrieveByID(ctx, notifier.ID)
	if err != nil {
		return err
	}

	if _, err := ns.things.Authorize(ctx, &protomfx.AuthorizeReq{Token: token, Object: nf.GroupID, Subject: things.GroupSub, Action: things.Viewer}); err != nil {
		return err
	}

	if err := ns.validateContacts(notifier.Contacts); err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return ns.notifierRepo.Update(ctx, notifier)
}

func (ns *notifierService) RemoveNotifiers(ctx context.Context, token, groupID string, ids ...string) error {
	if _, err := ns.things.Authorize(ctx, &protomfx.AuthorizeReq{Token: token, Object: groupID, Subject: things.GroupSub, Action: things.Editor}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := ns.notifierRepo.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}

func IsPhoneNumber(phoneNumber string) bool {
	// phoneRegexp represent regex pattern to validate E.164 phone numbers
	var phoneRegexp = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

	return phoneRegexp.MatchString(phoneNumber)
}

func (ns *notifierService) validateContacts(contacts []string) error {
	err := apiutil.ErrInvalidContact
	switch ns.svcName {
	case "smtp-notifier":
		for _, c := range contacts {
			if !email.IsEmail(c) {
				return err
			}
		}
		return nil
	case "smpp-notifier":
		for _, c := range contacts {
			if !IsPhoneNumber(c) {
				return err
			}
		}
		return nil
	default:
		return err
	}
}
