package dbutil

import "github.com/MainfluxLabs/mainflux/pkg/errors"

var (
	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrNotFound indicates that an entity was not found in the database.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that entity an entity with conflicting properties already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrCreateEntity indicates an error in attempting to create an entity or entities.
	ErrCreateEntity = errors.New("failed to create entity")

	// ErrRetrieveEntity indicates an error in attempting to retrieve an entity or entities.
	ErrRetrieveEntity = errors.New("failed to retrieve entity")

	// ErrUpdateEntity indicates an error in attempting to update an entity or entities.
	ErrUpdateEntity = errors.New("failed to update entity")

	// ErrRemoveEntity indicates an error in attempting to remove an entity or entities.
	ErrRemoveEntity = errors.New("failed to remove entity")

	// ErrScanMetadata indicates an error in attempting to decode entity metadata from the database.
	ErrScanMetadata = errors.New("failed to scan metadata from db")
)
