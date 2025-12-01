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

	// ErrMissingNotifierID indicates missing notifier ID.
	ErrMissingNotifierID = errors.New("missing notifier id")

	// ErrMissingAlarmID indicates missing alarm ID.
	ErrMissingAlarmID = errors.New("missing alarm id")

	// ErrMissingRuleID indicates missing rule ID.
	ErrMissingRuleID = errors.New("missing rule id")

	// ErrMissingUserID indicates missing user ID.
	ErrMissingUserID = errors.New("missing user id")

	// ErrMissingRole indicates missing role.
	ErrMissingRole = errors.New("missing role")

	// ErrMissingObject indicates missing object.
	ErrMissingObject = errors.New("missing object")

	// ErrMissingKeyID indicates missing ID of key.
	ErrMissingKeyID = errors.New("missing key ID")

	// ErrMissingExternalThingKey indicates missing external thing key
	ErrMissingExternalThingKey = errors.New("missing external thing key")

	// ErrMissingInviteID incidates missing ID of Invite.
	ErrMissingInviteID = errors.New("missing invite ID")

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

	// ErrMissingSerial indicates missing serial.
	ErrMissingSerial = errors.New("missing serial")

	// ErrMissingCertData indicates missing cert data (ttl, key_type or key_bits).
	ErrMissingCertData = errors.New("missing certificate data")

	// ErrInvalidContact indicates an invalid contact.
	ErrInvalidContact = errors.New("invalid contact")

	// ErrMissingEmail indicates missing email.
	ErrMissingEmail = errors.New("missing email")

	// ErrMissingEmailToken indicates a missing e-mail verification token.
	ErrMissingEmailToken = errors.New("missing e-mail verification token")

	// ErrMissingRedirectPath indicates missing endpoint path from the frontend.
	ErrMissingRedirectPath = errors.New("missing redirect path")

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

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrInvalidQueryParams indicates invalid query parameters
	ErrInvalidQueryParams = errors.New("invalid query parameters")

	// ErrInvalidAggType indicates invalid aggregation type
	ErrInvalidAggType = errors.New("invalid aggregation type")

	// ErrInvalidAggInterval indicates invalid aggregation interval
	ErrInvalidAggInterval = errors.New("invalid aggregation interval")

	// ErrNotFoundParam indicates that the parameter was not found in the query
	ErrNotFoundParam = errors.New("parameter not found in the query")

	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrInvalidRole indicates an invalid role.
	ErrInvalidRole = errors.New("invalid role")

	// ErrMissingConditionField indicates a missing condition field
	ErrMissingConditionField = errors.New("missing condition field")

	// ErrMissingConditionComparator indicates a missing condition operator
	ErrMissingConditionComparator = errors.New("missing condition comparator")

	// ErrMissingConditionThreshold indicates a missing condition threshold
	ErrMissingConditionThreshold = errors.New("missing condition threshold")

	// ErrInvalidActionType indicates an invalid action type
	ErrInvalidActionType = errors.New("missing or invalid action type")

	// ErrMissingActionID indicates a missing action id
	ErrMissingActionID = errors.New("missing action id")

	// ErrInvalidOperator indicates an invalid logical operator
	ErrInvalidOperator = errors.New("missing or invalid logical operator")

	// ErrInviteExpired indicates that an invite has expired
	ErrInviteExpired = errors.New("invite expired")

	// ErrInviteExpired indicates that an invite is in an invalid state for a certain action to be performed on it
	ErrInvalidInviteState = errors.New("invalid invite state")

	// ErrUserAlreadyInvited indicates that the invitee already has a pending invitation to join the same Org
	ErrUserAlreadyInvited = errors.New("user already has pending invite to org")

	// ErrInvalidThingKeyType indicates an invalid or missing type of thing authentication key
	ErrInvalidThingKeyType = errors.New("invalid thing key type")

	// ErrMissingProvider indicates an invalid or missing provider
	ErrMissingProvider = errors.New("missing or invalid provider")

	// ErrMissingProviderCode indicates an missing provider code
	ErrMissingProviderCode = errors.New("missing provider code")

	// ErrMissingState indicates an invalid or missing state
	ErrMissingState = errors.New("missing or invalid state")
)
