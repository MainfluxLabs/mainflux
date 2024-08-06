// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers"
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
	ListNotifiersByGroup(ctx context.Context, token string, groupID string) ([]things.Notifier, error)

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
	notifierRepo NotifierRepository
	things       protomfx.ThingsServiceClient
}

// New instantiates the subscriptions service implementation.
func New(idp uuid.IDProvider, notifier Notifier, from string, notifierRepo NotifierRepository, things protomfx.ThingsServiceClient) Service {
	return &notifierService{
		idp:          idp,
		notifier:     notifier,
		from:         from,
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
		nf, err := ns.createNotifier(ctx, &notifier, token)
		if err != nil {
			return []things.Notifier{}, err
		}
		nfs = append(nfs, nf)
	}

	return nfs, nil
}

func (ns *notifierService) createNotifier(ctx context.Context, notifier *things.Notifier, token string) (things.Notifier, error) {
	_, err := ns.things.CanAccessGroup(ctx, &protomfx.AccessGroupReq{Token: token, GroupID: notifier.GroupID, Action: things.Editor})
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

func (ns *notifierService) ListNotifiersByGroup(ctx context.Context, token string, groupID string) ([]things.Notifier, error) {
	_, err := ns.things.CanAccessGroup(ctx, &protomfx.AccessGroupReq{Token: token, GroupID: groupID, Action: things.Viewer})
	if err != nil {
		return []things.Notifier{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	notifiers, err := ns.notifierRepo.RetrieveByGroupID(ctx, groupID)
	if err != nil {
		return []things.Notifier{}, err
	}

	return notifiers, nil
}

func (ns *notifierService) ViewNotifier(ctx context.Context, token, id string) (things.Notifier, error) {
	notifier, err := ns.notifierRepo.RetrieveByID(ctx, id)
	if err != nil {
		return things.Notifier{}, err
	}

	if _, err := ns.things.CanAccessGroup(ctx, &protomfx.AccessGroupReq{Token: token, GroupID: notifier.GroupID, Action: things.Viewer}); err != nil {
		return things.Notifier{}, err
	}

	return notifier, nil
}

func (ns *notifierService) UpdateNotifier(ctx context.Context, token string, notifier things.Notifier) error {
	nf, err := ns.notifierRepo.RetrieveByID(ctx, notifier.ID)
	if err != nil {
		return err
	}

	if _, err := ns.things.CanAccessGroup(ctx, &protomfx.AccessGroupReq{Token: token, GroupID: nf.GroupID, Action: things.Viewer}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return ns.notifierRepo.Update(ctx, notifier)
}

func (ns *notifierService) RemoveNotifiers(ctx context.Context, token, groupID string, ids ...string) error {
	if _, err := ns.things.CanAccessGroup(ctx, &protomfx.AccessGroupReq{Token: token, GroupID: groupID, Action: things.Editor}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := ns.notifierRepo.Remove(ctx, groupID, ids...); err != nil {
		return err
	}

	return nil
}
