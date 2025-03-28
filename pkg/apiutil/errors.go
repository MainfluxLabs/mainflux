// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import "github.com/MainfluxLabs/mainflux/pkg/errors"

// Errors defined in this file are used by the LoggingErrorEncoder decorator
// to distinguish and log API request validation errors and avoid that service
// errors are logged twice.
var (
	// ErrBearerToken indicates missing or invalid bearer user token.
	ErrBearerToken = errors.New("missing or invalid bearer user token")

	// ErrBearerKey indicates missing or invalid bearer entity key.
	ErrBearerKey = errors.New("missing or invalid bearer entity key")

	// ErrMissingGroupID indicates missing group ID.
	ErrMissingGroupID = errors.New("missing group id")

	// ErrMissingOrgID indicates missing org ID.
	ErrMissingOrgID = errors.New("missing org id")

	// ErrMissingThingID indicates missing thing ID.
	ErrMissingThingID = errors.New("missing thing id")

	// ErrMissingProfileID indicates missing profile ID.
	ErrMissingProfileID = errors.New("missing profile id")

	// ErrMissingMemberID indicates missing member ID.
	ErrMissingMemberID = errors.New("missing member id")

	// ErrMissingWebhookID indicates missing webhook ID.
	ErrMissingWebhookID = errors.New("missing webhook id")

	// ErrMissingNotifierID indicates missing notifier ID.
	ErrMissingNotifierID = errors.New("missing notifier id")

	// ErrMissingAlarmID indicates missing alarm ID.
	ErrMissingAlarmID = errors.New("missing alarm id")

	// ErrMissingUserID indicates missing user ID.
	ErrMissingUserID = errors.New("missing user id")

	// ErrMissingRole indicates missing role.
	ErrMissingRole = errors.New("missing role")

	// ErrMissingObject indicates missing object.
	ErrMissingObject = errors.New("missing object")

	// ErrMissingKeyID indicates missing ID of key.
	ErrMissingKeyID = errors.New("missing key ID")

	// ErrInvalidSubject indicates invalid subject.
	ErrInvalidSubject = errors.New("invalid subject")

	// ErrInvalidAction indicates invalid action.
	ErrInvalidAction = errors.New("invalid action")

	// ErrInvalidAuthKey indicates invalid auth key.
	ErrInvalidAuthKey = errors.New("invalid auth key")

	// ErrInvalidIDFormat indicates an invalid ID format.
	ErrInvalidIDFormat = errors.New("invalid id format provided")

	// ErrNameSize indicates that name size exceeds the max.
	ErrNameSize = errors.New("invalid name size")

	// ErrEmailSize indicates that email size exceeds the max.
	ErrEmailSize = errors.New("invalid email size")

	// ErrInvalidStatus indicates an invalid user account status.
	ErrInvalidStatus = errors.New("invalid user account status")

	// ErrLimitSize indicates that an invalid limit.
	ErrLimitSize = errors.New("invalid limit size")

	// ErrOffsetSize indicates an invalid offset.
	ErrOffsetSize = errors.New("invalid offset size")

	// ErrInvalidOrder indicates an invalid list order.
	ErrInvalidOrder = errors.New("invalid list order provided")

	// ErrInvalidDirection indicates an invalid list direction.
	ErrInvalidDirection = errors.New("invalid list direction provided")

	// ErrEmptyList indicates that entity data is empty.
	ErrEmptyList = errors.New("empty list provided")

	// ErrMissingCertID indicates missing cert ID.
	ErrMissingCertID = errors.New("missing cert id")

	// ErrMissingCertData indicates missing cert data (ttl, key_type or key_bits).
	ErrMissingCertData = errors.New("missing certificate data")

	// ErrInvalidTopic indicates an invalid subscription topic.
	ErrInvalidTopic = errors.New("invalid Subscription topic")

	// ErrInvalidContact indicates an invalid contact.
	ErrInvalidContact = errors.New("invalid contact")

	// ErrMissingEmail indicates missing email.
	ErrMissingEmail = errors.New("missing email")

	// ErrMissingHost indicates missing host.
	ErrMissingHost = errors.New("missing host")

	// ErrMissingPass indicates missing password.
	ErrMissingPass = errors.New("missing password")

	// ErrMissingConfPass indicates missing conf password.
	ErrMissingConfPass = errors.New("missing conf password")

	// ErrInvalidResetPass indicates an invalid reset password.
	ErrInvalidResetPass = errors.New("invalid reset password")

	// ErrInvalidComparator indicates an invalid comparator.
	ErrInvalidComparator = errors.New("invalid comparator")

	// ErrInvalidAPIKey indicates an invalid API key type.
	ErrInvalidAPIKey = errors.New("invalid api key type")

	// ErrMaxLevelExceeded indicates an invalid group level.
	ErrMaxLevelExceeded = errors.New("invalid group level (should be lower than 5)")

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrInvalidQueryParams indicates invalid query parameters
	ErrInvalidQueryParams = errors.New("invalid query parameters")

	// ErrNotFoundParam indicates that the parameter was not found in the query
	ErrNotFoundParam = errors.New("parameter not found in the query")

	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrInvalidRole indicates an invalid role.
	ErrInvalidRole = errors.New("invalid role")
)
