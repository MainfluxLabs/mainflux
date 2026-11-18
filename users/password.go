// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	_ "embed"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"golang.org/x/text/unicode/norm"
)

const (
	// MinPasswordLength is the lowest allowed value of the configured minimum password length.
	MinPasswordLength = 8
	// MaxPasswordLength is the maximum password length in Unicode code points.
	MaxPasswordLength = 128

	minEmailLocalPartLength = 3
)

var (
	// ErrPasswordTooShort indicates that the password is shorter than the minimum length.
	ErrPasswordTooShort = errors.New("password is too short")

	// ErrPasswordTooLong indicates that the password is longer than the maximum length.
	ErrPasswordTooLong = errors.New("password is too long")

	// ErrCommonPassword indicates that the password is on the list of common passwords.
	ErrCommonPassword = errors.New("password is too common")

	// ErrPasswordContainsEmail indicates that the password contains the user's e-mail local part.
	ErrPasswordContainsEmail = errors.New("password must not contain the e-mail address")
)

// NCSC list of the 100,000 most used passwords, from SecLists.
//
//go:embed common_passwords.txt
var commonPasswordsList string

var commonPasswords = sync.OnceValue(func() map[string]struct{} {
	passwords := make(map[string]struct{})
	for _, password := range strings.Split(commonPasswordsList, "\n") {
		password = strings.ToLower(NormalizePassword(strings.TrimSpace(password)))
		// Shorter entries can never pass the length check.
		if utf8.RuneCountInString(password) < MinPasswordLength {
			continue
		}
		passwords[password] = struct{}{}
	}
	return passwords
})

// NormalizePassword returns the password in Unicode NFKC form, used for validation and hashing.
func NormalizePassword(password string) string {
	return norm.NFKC.String(password)
}

// ValidatePassword returns an error if the password does not meet the password policy.
func ValidatePassword(password, email string, minLength int) error {
	password = NormalizePassword(password)
	minLength = max(minLength, MinPasswordLength)

	length := utf8.RuneCountInString(password)
	if length < minLength {
		return errors.Wrap(ErrPasswordTooShort, errors.ErrPasswordFormat)
	}
	if length > MaxPasswordLength {
		return errors.Wrap(ErrPasswordTooLong, errors.ErrPasswordFormat)
	}

	lowerPassword := strings.ToLower(password)
	if _, ok := commonPasswords()[lowerPassword]; ok {
		return errors.Wrap(ErrCommonPassword, errors.ErrPasswordFormat)
	}

	localPart := email
	if at := strings.LastIndex(email, "@"); at >= 0 {
		localPart = email[:at]
	}
	localPart = strings.ToLower(NormalizePassword(localPart))
	if utf8.RuneCountInString(localPart) >= minEmailLocalPartLength && strings.Contains(lowerPassword, localPart) {
		return errors.Wrap(ErrPasswordContainsEmail, errors.ErrPasswordFormat)
	}

	return nil
}
