// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package errors

var (
	// ErrAuthentication indicates failure occurred while authenticating the entity.
	ErrAuthentication = New("failed to perform authentication over the entity")

	// ErrAuthorization indicates failure occurred while authorizing the entity.
	ErrAuthorization = New("failed to perform authorization over the entity")

	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = New("malformed entity specification")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = New("entity not found")

	// ErrConflict indicates that entity already exists.
	ErrConflict = New("entity already exists")

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = New("failed to create entity in the db")

	// ErrRetrieveEntity indicates error in viewing entity or entities.
	ErrRetrieveEntity = New("failed to retrieve entity")

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = New("failed to update entity")

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = New("failed to remove entity")

	// ErrScanMetadata indicates problem with metadata in db.
	ErrScanMetadata = New("failed to scan metadata in db")

	// ErrSaveMessage indicates failure occurred while saving message to database.
	ErrSaveMessage = New("failed to save message to database")

	// ErrMessage indicates an error converting a message to Mainflux message.
	ErrMessage = New("failed to convert to Mainflux message")
)
