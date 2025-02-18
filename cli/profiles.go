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
		Use:   "get <all | by-thing | by-id> <id> <user_token>",
		Short: "Get profile",
		Long: `Get all profiles, get profile by thing or get profile by id. Profiles can be filtered by name or metadata.
		all - lists all profiles
		by-thing - shows profile by thing with provided <id>
		by-id - shows profile with provided <id>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 && len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			metadata, err := convertMetadata(Metadata)
			if err != nil {
				logError(err)
				return
			}
			pageMetadata := mfxsdk.PageMetadata{
				Name:     Name,
				Offset:   uint64(Offset),
				Limit:    uint64(Limit),
				Metadata: metadata,
			}

			switch args[0] {
			case "all":
				l, err := sdk.Profiles(args[1], pageMetadata)
				if err != nil {
					logError(err)
					return
				}

				logJSON(l)
				return
			case "by-thing":
				pbt, err := sdk.ViewProfileByThing(args[1], args[2])
				if err != nil {
					logError(err)
					return
				}

				logJSON(pbt)
				return
			case "by-id":
				c, err := sdk.Profile(args[1], args[2])
				if err != nil {
					logError(err)
					return
				}

				logJSON(c)
				return
			default:
				logUsage(cmd.Use)
				return
			}
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
}

// NewProfilesCmd returns profiles command.
func NewProfilesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "profiles [create | get | update | delete]",
		Short: "Profiles management",
		Long:  `Profiles management: create, get, update or delete Profile, get Profile by Thing`,
	}

	for i := range cmdProfiles {
		cmd.AddCommand(&cmdProfiles[i])
	}

	return &cmd
}
