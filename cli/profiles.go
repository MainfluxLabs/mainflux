// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdProfiles = []cobra.Command{
	{
		Use:   "create <JSON_profile> <group_id> <user_token>",
		Short: "Create profile",
		Long:  `Creates new profile and generates it's UUID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var profile mfxsdk.Profile
			if err := json.Unmarshal([]byte(args[0]), &profile); err != nil {
				logError(err)
				return
			}

			id, err := sdk.CreateProfile(profile, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	{
		Use:   "get [all | <profile_id>] <user_token>",
		Short: "Get profile",
		Long: `Get all profiles or get profile by id. Profiles can be filtered by name or metadata.
		all - lists all profiles
		<profile_id> - shows thing with provided <profile_id>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			metadata, err := convertMetadata(Metadata)
			if err != nil {
				logError(err)
				return
			}
			pageMetadata := mfxsdk.PageMetadata{
				Name:     "",
				Offset:   uint64(Offset),
				Limit:    uint64(Limit),
				Metadata: metadata,
			}

			if args[0] == "all" {
				l, err := sdk.Profiles(args[1], pageMetadata)
				if err != nil {
					logError(err)
					return
				}

				logJSON(l)
				return
			}
			c, err := sdk.Profile(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
	{
		Use:   "update <JSON_string> <profile_id> <user_token>",
		Short: "Update profile",
		Long:  `Updates profile record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var profile mfxsdk.Profile
			if err := json.Unmarshal([]byte(args[0]), &profile); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateProfile(profile, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "delete <profile_id> <user_token>",
		Short: "Delete profile",
		Long:  `Delete profile by ID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.DeleteProfile(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "connections <profile_id> <user_token>",
		Short: "Connections list",
		Long:  `List of Things connected to a Profile`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			cl, err := sdk.ThingsByProfile(args[1], args[0], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}

			logJSON(cl)
		},
	},
}

// NewProfilesCmd returns profiles command.
func NewProfilesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "profiles [create | get | update | delete | connections]",
		Short: "Profiles management",
		Long:  `Profiles management: create, get, update or delete Profile and get list of Things connected to a Profile`,
	}

	for i := range cmdProfiles {
		cmd.AddCommand(&cmdProfiles[i])
	}

	return &cmd
}
