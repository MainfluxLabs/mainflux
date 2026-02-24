package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveDownlinks            = "save_downlinks"
	retrieveDownlinksByThing = "retrieve_downlinks_by_thing"
	retrieveDownlinksByGroup = "retrieve_downlinks_by_group"
	retrieveDownlinkByID     = "retrieve_downlink_by_id"
	retrieveAllDownlinks     = "retrieve_all_downlinks"
	updateDownlink           = "update_downlink"
	removeDownlinks          = "remove_downlinks"
	removeDownlinksByThing   = "remove_downlinks_by_thing"
	removeDownlinksByGroup   = "remove_downlinks_by_group"
)

var (
	_ downlinks.DownlinkRepository = (*downlinkRepositoryMiddleware)(nil)
)

type downlinkRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   downlinks.DownlinkRepository
}

// DownlinkRepositoryMiddleware tracks request and their latency, and adds spans to context.
func DownlinkRepositoryMiddleware(tracer opentracing.Tracer, repo downlinks.DownlinkRepository) downlinks.DownlinkRepository {
	return downlinkRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (drm downlinkRepositoryMiddleware) Save(ctx context.Context, dls ...downlinks.Downlink) ([]downlinks.Downlink, error) {
	span := dbutil.CreateSpan(ctx, drm.tracer, saveDownlinks)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.Save(ctx, dls...)
}

func (drm downlinkRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	span := dbutil.CreateSpan(ctx, drm.tracer, retrieveDownlinksByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (drm downlinkRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	span := dbutil.CreateSpan(ctx, drm.tracer, retrieveDownlinksByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (drm downlinkRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (downlinks.Downlink, error) {
	span := dbutil.CreateSpan(ctx, drm.tracer, retrieveDownlinkByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.RetrieveByID(ctx, id)
}

func (drm downlinkRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]downlinks.Downlink, error) {
	span := dbutil.CreateSpan(ctx, drm.tracer, retrieveAllDownlinks)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.RetrieveAll(ctx)
}

func (drm downlinkRepositoryMiddleware) Update(ctx context.Context, d downlinks.Downlink) error {
	span := dbutil.CreateSpan(ctx, drm.tracer, updateDownlink)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.Update(ctx, d)
}

func (drm downlinkRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, drm.tracer, removeDownlinks)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.Remove(ctx, ids...)
}

func (drm downlinkRepositoryMiddleware) RemoveByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, drm.tracer, removeDownlinksByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.RemoveByThing(ctx, thingID)
}

func (drm downlinkRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, drm.tracer, removeDownlinksByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return drm.repo.RemoveByGroup(ctx, groupID)
}
