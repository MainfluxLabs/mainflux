// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package errors

import "errors"

var (
	// ErrAuthentication indicates failure occurred while authenticating the entity.
	ErrAuthentication = New("failed to perform authentication over the entity")

	// ErrAuthorization indicates failure occurred while authorizing the entity.
	ErrAuthorization = New("failed to perform authorization over the entity")

	// ErrSaveMessages indicates failure occurred while saving messages to database.
	ErrSaveMessages = New("failed to save messages to database")

	// ErrDeleteMessage indicates failure occurred while deleting messages in the database.
	ErrDeleteMessages = New("failed to delete messages")

	// ErrInvalidMessage indicates that message format is invalid.
	ErrInvalidMessage = errors.New("invalid message representation")

	// ErrTransRollback indicates failure occured while trying to rollback transaction.
	ErrTxRollback = errors.New("failed to rollback transaction")

	// ErrBackupMessages indicates failure occurred while backing up messages from the database.
	ErrBackupMessages = New("failed to backup messages")

	// ErrRestoreMessages indicates failure occured while restoring messages to the database.
	ErrRestoreMessages = New("failed to restore messages")

	// ErrMessage indicates an error converting a message to Mainflux message.
	ErrMessage = New("failed to convert to Mainflux message")

	// ErrInvalidPassword indicates that current password is invalid.
	ErrInvalidPassword = New("invalid current password")
)
