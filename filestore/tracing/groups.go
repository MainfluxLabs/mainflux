package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroupFile      = "save_group_file"
	updateGroupFile    = "update_group_file"
	retrieveGroupFile  = "retrieve_group_file"
	retrieveGroupFiles = "retrieve_group_files"
	removeGroupFile    = "remove_group_file"
	removeGroupFiles   = "remove_group_files"
)

var (
	_ filestore.GroupsRepository = (*groupsRepositoryMiddleware)(nil)
)

type groupsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   filestore.GroupsRepository
}

// GroupsRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func GroupsRepositoryMiddleware(tracer opentracing.Tracer, repo filestore.GroupsRepository) filestore.GroupsRepository {
	return groupsRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (grm groupsRepositoryMiddleware) Save(ctx context.Context, groupID string, fi filestore.FileInfo) error {
	span := dbutil.CreateSpan(ctx, grm.tracer, saveGroupFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Save(ctx, groupID, fi)
}

func (grm groupsRepositoryMiddleware) Update(ctx context.Context, groupID string, fi filestore.FileInfo) error {
	span := dbutil.CreateSpan(ctx, grm.tracer, updateGroupFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Update(ctx, groupID, fi)
}

func (grm groupsRepositoryMiddleware) Retrieve(ctx context.Context, groupID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveGroupFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Retrieve(ctx, groupID, fi)
}

func (grm groupsRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, fi filestore.FileInfo, pm filestore.PageMetadata) (filestore.FileGroupsPage, error) {
	span := dbutil.CreateSpan(ctx, grm.tracer, retrieveGroupFiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByGroup(ctx, groupID, fi, pm)
}

func (grm groupsRepositoryMiddleware) Remove(ctx context.Context, groupID string, fi filestore.FileInfo) error {
	span := dbutil.CreateSpan(ctx, grm.tracer, removeGroupFile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Remove(ctx, groupID, fi)
}

func (grm groupsRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, grm.tracer, removeGroupFiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RemoveByGroup(ctx, groupID)
}
