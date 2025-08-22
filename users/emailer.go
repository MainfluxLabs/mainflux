// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

// Emailer wrapper around the email
type Emailer interface {
	SendPasswordReset(To []string, redirectPath, token string) error
	SendEmailVerification(To []string, redirectPath, token string) error
	SendPlatformInvite(To []string, inv PlatformInvite, redirectPath string) error
}
