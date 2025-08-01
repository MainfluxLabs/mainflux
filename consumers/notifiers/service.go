// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service represents a notification service.
type Service interface {
	// CreateNotifiers creates notifiers for certain group identified by the provided ID
	CreateNotifiers(ctx context.Context, token string, notifiers ...Notifier) ([]Notifier, error)

	// ListNotifiersByGroup retrieves data about a subset of notifiers
	// related to a certain group identified by the provided ID.
	ListNotifiersByGroup(ctx context.Context, token string, groupID string, pm apiutil.PageMetadata) (NotifiersPage, error)

	// ViewNotifier retrieves data about the notifier identified with the provided ID
	ViewNotifier(ctx context.Context, token, id string) (Notifier, error)

	// UpdateNotifier updates the notifier identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateNotifier(ctx context.Context, token string, notifier Notifier) error

	// RemoveNotifiers removes the notifiers identified with the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveNotifiers(ctx context.Context, token string, id ...string) error

	consumers.Consumer
}

const (
	notifierSMTP = "smtp"
	notifierSMPP = "smpp"
)

var _ Service = (*notifierService)(nil)

type notifierService struct {
	idp          uuid.IDProvider
	sender       Sender
	notifierRepo NotifierRepository
	things       protomfx.ThingsServiceClient
}

// New instantiates the subscriptions service implementation.
func New(idp uuid.IDProvider, sender Sender, notifierRepo NotifierRepository, things protomfx.ThingsServiceClient) Service {
	return &notifierService{
		idp:          idp,
		sender:       sender,
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

	subject := strings.Split(msg.Subject, ".")
	notifierType := subject[0]
	notifierID := subject[1]

	switch notifierType {
	case notifierSMTP:
		smtp, err := ns.notifierRepo.RetrieveByID(ctx, notifierID)
		if err != nil {
			return errors.Wrap(ErrNotify, err)
		}
		if err = ns.sender.Send(smtp.Contacts, msg); err != nil {
			return err
		}
	case notifierSMPP:
		smpp, err := ns.notifierRepo.RetrieveByID(ctx, notifierID)
		if err != nil {
			return errors.Wrap(ErrNotify, err)
		}
		if err = ns.sender.Send(smpp.Contacts, msg); err != nil {
			return err
		}
	}

	return nil
}

func (ns *notifierService) CreateNotifiers(ctx context.Context, token string, notifiers ...Notifier) ([]Notifier, error) {
	nfs := []Notifier{}
	for _, notifier := range notifiers {
		if err := ns.sender.ValidateContacts(notifier.Contacts); err != nil {
			return []Notifier{}, errors.Wrap(errors.ErrMalformedEntity, err)
		}

		nf, err := ns.createNotifier(ctx, &notifier, token)
		if err != nil {
			return []Notifier{}, err
		}
		nfs = append(nfs, nf)
	}

	return nfs, nil
}

func (ns *notifierService) createNotifier(ctx context.Context, notifier *Notifier, token string) (Notifier, error) {
	_, err := ns.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: notifier.GroupID, Action: things.Editor})
	if err != nil {
		return Notifier{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	id, err := ns.idp.ID()
	if err != nil {
		return Notifier{}, err
	}
	notifier.ID = id

	nfs, err := ns.notifierRepo.Save(ctx, *notifier)
	if err != nil {
		return Notifier{}, err
	}

	if len(nfs) == 0 {
		return Notifier{}, errors.ErrCreateEntity
	}

	return nfs[0], nil
}

func (ns *notifierService) ListNotifiersByGroup(ctx context.Context, token string, groupID string, pm apiutil.PageMetadata) (NotifiersPage, error) {
	_, err := ns.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer})
	if err != nil {
		return NotifiersPage{}, err
	}

	notifiers, err := ns.notifierRepo.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return NotifiersPage{}, err
	}

	return notifiers, nil
}

func (ns *notifierService) ViewNotifier(ctx context.Context, token, id string) (Notifier, error) {
	notifier, err := ns.notifierRepo.RetrieveByID(ctx, id)
	if err != nil {
		return Notifier{}, err
	}

	if _, err := ns.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: notifier.GroupID, Action: things.Viewer}); err != nil {
		return Notifier{}, err
	}

	return notifier, nil
}

func (ns *notifierService) UpdateNotifier(ctx context.Context, token string, notifier Notifier) error {
	nf, err := ns.notifierRepo.RetrieveByID(ctx, notifier.ID)
	if err != nil {
		return err
	}

	if _, err := ns.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: nf.GroupID, Action: things.Viewer}); err != nil {
		return err
	}

	if err := ns.sender.ValidateContacts(notifier.Contacts); err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return ns.notifierRepo.Update(ctx, notifier)
}

func (ns *notifierService) RemoveNotifiers(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		notifier, err := ns.notifierRepo.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		if _, err := ns.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: notifier.GroupID, Action: things.Editor}); err != nil {
			return errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	if err := ns.notifierRepo.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}
