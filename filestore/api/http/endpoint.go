// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/go-kit/kit/endpoint"
)

func saveFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(saveFileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.SaveFile(ctx, req.file, req.key.Value, req.fileInfo)
		if err != nil {
			return nil, err
		}

		return fileRes{created: true}, nil
	}
}

func updateFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateFileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.UpdateFile(ctx, req.key.Value, req.fileInfo)
		if err != nil {
			return nil, err
		}

		return fileRes{created: false}, nil
	}
}

func listFilesEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listFilesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Class:  req.class,
			Format: req.format,
			Name:   req.name,
		}

		page, err := svc.ListFiles(ctx, req.key.Value, fi, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildListFilesResponse(page.PageMetadata, page.Files), nil
	}
}

func viewFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(fileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Name:   req.name,
			Class:  req.class,
			Format: req.format,
		}

		f, err := svc.ViewFile(ctx, req.key.Value, fi)
		if err != nil {
			return nil, err
		}

		res := viewFileRes{
			file: f,
		}

		return res, nil
	}
}

func removeFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(fileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Name:   req.name,
			Class:  req.class,
			Format: req.format,
		}

		err := svc.RemoveFile(ctx, req.key.Value, fi)
		if err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func saveGroupFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(saveGroupFileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.SaveGroupFile(ctx, req.file, req.token, req.groupID, req.fileInfo)
		if err != nil {
			return nil, err
		}

		return fileRes{created: true}, nil
	}
}

func updateGroupFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateGroupFileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.UpdateGroupFile(ctx, req.token, req.groupID, req.fileInfo)
		if err != nil {
			return nil, err
		}

		return fileRes{created: false}, nil
	}
}

func listGroupFilesEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listGroupFilesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Class:  req.class,
			Format: req.format,
			Name:   req.name,
		}

		page, err := svc.ListGroupFiles(ctx, req.token, req.groupID, fi, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildListFilesResponse(page.PageMetadata, page.Files), nil
	}
}

func viewGroupFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(groupFileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Name:   req.name,
			Class:  req.class,
			Format: req.format,
		}

		f, err := svc.ViewGroupFile(ctx, req.token, req.groupID, fi)
		if err != nil {
			return nil, err
		}

		res := viewFileRes{
			file: f,
		}

		return res, nil
	}
}

func viewGroupFileByKeyEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(groupFileByKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Name:   req.name,
			Class:  req.class,
			Format: req.format,
		}

		f, err := svc.ViewGroupFileByKey(ctx, req.key.Value, fi)
		if err != nil {
			return nil, err
		}

		res := viewFileRes{
			file: f,
		}

		return res, nil
	}
}

func removeGroupFileEndpoint(svc filestore.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(groupFileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		fi := filestore.FileInfo{
			Name:   req.name,
			Class:  req.class,
			Format: req.format,
		}

		err := svc.RemoveGroupFile(ctx, req.token, req.groupID, fi)
		if err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildListFilesResponse(pm filestore.PageMetadata, files []filestore.FileInfo) listFilesRes {
	res := listFilesRes{
		pageRes: pageRes{
			Total:  pm.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		FilesInfo: []fileInfo{},
	}
	for _, file := range files {
		f := fileInfo{
			Name:     file.Name,
			Class:    file.Class,
			Format:   file.Format,
			Time:     file.Time,
			Metadata: file.Metadata,
		}

		res.FilesInfo = append(res.FilesInfo, f)
	}

	return res
}
