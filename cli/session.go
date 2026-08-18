// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var errNoSession = errors.New("no saved session; run \"users token <email> <password>\" first")

const (
	sessionDirName  = ".mainflux"
	sessionFileName = "session.json"

	// sessionFileMode keeps the stored tokens readable only by their owner.
	sessionFileMode = 0o600
	sessionDirMode  = 0o700
)

// SessionTokenArg is the placeholder a user can pass in place of a token
// argument to mean "use the token saved by `users token`".
const SessionTokenArg = "-"

func sessionPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, sessionDirName, sessionFileName), nil
}

// SaveSession persists the token pair so later commands can reuse it.
func SaveSession(tokens mfxsdk.TokenPair) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), sessionDirMode); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, sessionFileMode)
}

// LoadSession reads the saved token pair. A missing file is not an error: it
// simply means no session has been established yet.
func LoadSession() (mfxsdk.TokenPair, error) {
	path, err := sessionPath()
	if err != nil {
		return mfxsdk.TokenPair{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mfxsdk.TokenPair{}, nil
		}
		return mfxsdk.TokenPair{}, err
	}

	var tokens mfxsdk.TokenPair
	if err := json.Unmarshal(data, &tokens); err != nil {
		return mfxsdk.TokenPair{}, err
	}

	return tokens, nil
}

// ClearSession removes the stored token pair.
func ClearSession() error {
	path, err := sessionPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// ResolveToken maps a token command argument to the token actually used. An
// empty argument or the SessionTokenArg placeholder falls back to the saved
// session, so users need not paste a token into every command.
func ResolveToken(arg string) string {
	if arg != "" && arg != SessionTokenArg {
		return arg
	}

	tokens, err := LoadSession()
	if err != nil {
		return arg
	}

	return tokens.AccessToken
}

var cmdSession = []cobra.Command{
	{
		Use:   "show",
		Short: "Show saved session",
		Long:  `Print the token pair saved by "users token".`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			tokens, err := LoadSession()
			if err != nil {
				logError(err)
				return
			}
			if tokens.AccessToken == "" {
				logError(errNoSession)
				return
			}

			logJSON(tokens)
		},
	},
	{
		Use:   "refresh",
		Short: "Refresh saved session",
		Long:  `Exchange the saved refresh token for a new access token and store it.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			tokens, err := LoadSession()
			if err != nil {
				logError(err)
				return
			}
			if tokens.RefreshToken == "" {
				logError(errNoSession)
				return
			}

			refreshed, err := sdk.RefreshToken(tokens.RefreshToken)
			if err != nil {
				logError(err)
				return
			}

			if err := SaveSession(refreshed); err != nil {
				logError(err)
				return
			}

			logJSON(refreshed)
		},
	},
	{
		Use:   "clear",
		Short: "Clear saved session",
		Long:  `Remove the locally saved token pair. Refresh tokens are stateless, so this does not revoke them server-side.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			if err := ClearSession(); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewSessionCmd returns session command.
func NewSessionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "session [show | refresh | clear]",
		Short: "Session management",
		Long:  `Session management: inspect, refresh, or clear the locally saved token pair.`,
	}

	for i := range cmdSession {
		cmd.AddCommand(&cmdSession[i])
	}

	return &cmd
}
