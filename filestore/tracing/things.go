package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveThingFile           = "save_thing_file"
	updateThingFile         = "update_thing_file"
	retrieveThingFile       = "retrieve_thing_file"
	retrieveThingFiles      = "retrieve_thing_files"
	removeThingFile         = "remove_thing_file"
	removeThingFiles        = "remove_thing_files"
	removeThingFilesByGroup = "remove_thing_files_by_group"

	retrieveThingIDsByGroup = "retrieve_thing_ids_by_group"
)

var (
	_ filestore.ThingsRepository = (*thingsRepositoryMiddleware)(nil)
)

type thingsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   filestore.ThingsRepository
}

// ThingsRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func ThingsRepositoryMiddleware(tracer opentracing.Tracer, repo filestore.ThingsRepository) filestore.ThingsRepository {
	return thingsRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm thingsRepositoryMiddleware) Save(ctx context.Context, thingID, groupID string, fi filestore.FileInfo) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, saveThingFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Save(ctx, thingID, groupID, fi)
}

func (trm thingsRepositoryMiddleware) Update(ctx context.Context, thingID string, fi filestore.FileInfo) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, updateThingFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, thingID, fi)
}

func (trm thingsRepositoryMiddleware) Retrieve(ctx context.Context, thingID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Retrieve(ctx, thingID, fi)
}

func (trm thingsRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, fi filestore.FileInfo, pm filestore.PageMetadata) (filestore.FileThingsPage, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingFiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByThing(ctx, thingID, fi, pm)
}

func (trm thingsRepositoryMiddleware) Remove(ctx context.Context, thingID string, fi filestore.FileInfo) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, removeThingFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Remove(ctx, thingID, fi)
}

func (trm thingsRepositoryMiddleware) RemoveByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, removeThingFiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RemoveByThing(ctx, thingID)
}

func (trm thingsRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, removeThingFilesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RemoveByGroup(ctx, groupID)
}

func (trm thingsRepositoryMiddleware) RetrieveThingIDsByGroup(ctx context.Context, groupID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingIDsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveThingIDsByGroup(ctx, groupID)
}
