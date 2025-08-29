package dbutil

import "github.com/MainfluxLabs/mainflux/pkg/errors"

var (
	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = errors.New("failed to create entity in the db")

	// ErrRetrieveEntity indicates error in viewing entity or entities.
	ErrRetrieveEntity = errors.New("failed to retrieve entity")

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = errors.New("failed to update entity")

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = errors.New("failed to remove entity")

	// ErrScanMetadata indicates problem with metadata in db.
	ErrScanMetadata = errors.New("failed to scan metadata in db")
)
