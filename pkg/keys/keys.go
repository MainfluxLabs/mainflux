// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys

const (
	// LoginKey is a temporary user key received on successful login.
	LoginKey uint32 = iota
	// RecoveryKey is a key used for resetting a password.
	RecoveryKey
	// APIKey enables acting on behalf of the user.
	APIKey
)
