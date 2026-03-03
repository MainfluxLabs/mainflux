package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveClients            = "save_clients"
	retrieveClientsByThing = "retrieve_clients_by_thing"
	retrieveClientsByGroup = "retrieve_clients_by_group"
	retrieveClientByID     = "retrieve_client_by_id"
	retrieveAllClients     = "retrieve_all_clients"
	updateClient           = "update_client"
	removeClients          = "remove_clients"
	removeClientsByThing   = "remove_clients_by_thing"
	removeClientsByGroup   = "remove_clients_by_group"
)

var _ modbus.ClientRepository = (*clientRepositoryMiddleware)(nil)

type clientRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   modbus.ClientRepository
}

// ClientRepositoryMiddleware tracks request and their latency, and adds spans to context.
func ClientRepositoryMiddleware(tracer opentracing.Tracer, repo modbus.ClientRepository) modbus.ClientRepository {
	return clientRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (crm clientRepositoryMiddleware) Save(ctx context.Context, cls ...modbus.Client) ([]modbus.Client, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, saveClients)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, cls...)
}

func (crm clientRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveClientsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (crm clientRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveClientsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (crm clientRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (modbus.Client, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveClientByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByID(ctx, id)
}

func (crm clientRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]modbus.Client, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveAllClients)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx)
}

func (crm clientRepositoryMiddleware) Update(ctx context.Context, c modbus.Client) error {
	span := dbutil.CreateSpan(ctx, crm.tracer, updateClient)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, c)
}

func (crm clientRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, crm.tracer, removeClients)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, ids...)
}

func (crm clientRepositoryMiddleware) RemoveByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, crm.tracer, removeClientsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RemoveByThing(ctx, thingID)
}

func (crm clientRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, crm.tracer, removeClientsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RemoveByGroup(ctx, groupID)
}
