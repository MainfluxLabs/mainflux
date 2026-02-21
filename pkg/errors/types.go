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

	// ErrDeleteMessages ErrDeleteMessage indicates failure occurred while deleting messages in the database.
	ErrDeleteMessages = New("failed to delete messages")

	// ErrInvalidMessage indicates that message format is invalid.
	ErrInvalidMessage = errors.New("invalid message representation")

	// ErrBackupMessages indicates failure occurred while backing up messages from the database.
	ErrBackupMessages = New("failed to backup messages")

	// ErrRestoreMessages indicates failure occured while restoring messages to the database.
	ErrRestoreMessages = New("failed to restore messages")

	// ErrMessage indicates an error converting a message to Mainflux message.
	ErrMessage = New("failed to convert to Mainflux message")

	// ErrBackupAlarms indicates failure occurred while backing up alarms from the database.
	ErrBackupAlarms = New("failed to backup alarms")

	// ErrInvalidPassword indicates that current password is invalid.
	ErrInvalidPassword = New("invalid current password")

	// ErrInvalidPayload indicates that a message payload could not be parsed.
	ErrInvalidPayload = New("invalid payload")

	// ErrInvalidSubject indicates that a message subject is malformed.
	ErrInvalidSubject = New("invalid subject")
)
