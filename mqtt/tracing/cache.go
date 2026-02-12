package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt/redis/cache"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveThingClient           = "save_thing_client"
	retrieveThingIDByClientID = "retrieve_thing_id_by_client_id"
	removeThingClient         = "remove_thing_client"
	removeThingClients        = "remove_thing_clients"
)

var _ cache.ConnectionCache = (*connectionCacheMiddleware)(nil)

type connectionCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  cache.ConnectionCache
}

// ConnectionCacheMiddleware tracks request and their latency, and adds spans to context.
func ConnectionCacheMiddleware(tracer opentracing.Tracer, cache cache.ConnectionCache) cache.ConnectionCache {
	return connectionCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (ccm connectionCacheMiddleware) Connect(ctx context.Context, clientID, thingID string) error {
	span := dbutil.CreateSpan(ctx, ccm.tracer, saveThingClient)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Connect(ctx, clientID, thingID)
}

func (ccm connectionCacheMiddleware) RetrieveThingByClient(ctx context.Context, clientID string) string {
	span := dbutil.CreateSpan(ctx, ccm.tracer, retrieveThingIDByClientID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.RetrieveThingByClient(ctx, clientID)
}

func (ccm connectionCacheMiddleware) Disconnect(ctx context.Context, clientID string) error {
	span := dbutil.CreateSpan(ctx, ccm.tracer, removeThingClient)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Disconnect(ctx, clientID)
}

func (ccm connectionCacheMiddleware) DisconnectByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, ccm.tracer, removeThingClients)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.DisconnectByThing(ctx, thingID)
}
