// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroups = []cobra.Command{
	{
		Use:   "create <JSON_group> <org_id> <user_token>",
		Short: "Create group",
		Long: `Creates new group:
		{
			"Name":<group_name>,
			"Description":<description>,
			"Metadata":<metadata>,
		}
		Name - is unique group name
		Metadata - JSON structured string`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			var group mfxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logError(err)
				return
			}
			id, err := sdk.CreateGroup(group, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logCreated(id)
		},
	},
	{
		Use:   "get <all | group-id | by-org> <id> <user_token>",
		Short: "Get group(s)",
		Long: `Get all groups, a specific group by ID, or all groups belonging to a specific Org.
		all - lists all groups
		group-id - shows group with provided <group-id>
		by-org - lists all groups belonging to Org with provided <org-id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsage(cmd.Use)
				return
			}

			switch args[0] {
			case "all":
				meta := mfxsdk.PageMetadata{
					Offset: uint64(Offset),
					Limit:  uint64(Limit),
				}
				gp, err := sdk.GetGroups(meta, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(gp)
			case "group-id":
				g, err := sdk.GetGroup(args[1], args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(g)
			case "by-org":
				if len(args) < 3 {
					logUsage(cmd.Use)
					return
				}

				res, err := sdk.GetGroupsByOrg(
					args[1],
					mfxsdk.PageMetadata{Offset: uint64(Offset), Limit: uint64(Limit)},
					args[2],
				)

				if err != nil {
					logError(err)
					return
				}

				logJSON(res)
			}
		},
	},
	{
		Use:   "delete <group_id> <user_token>",
		Short: "Delete group",
		Long:  `Delete group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			if err := sdk.DeleteGroup(args[0], args[1]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "update <JSON_group> <group_id> <user_token>",
		Short: "Update group",
		Long:  `Update group record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			var group mfxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logError(err)
				return
			}
			if err := sdk.UpdateGroup(group, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "things <group_id> <user_token>",
		Short: "Things by group",
		Long:  `Lists all things of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			meta := mfxsdk.PageMetadata{
				Offset: uint64(Offset),
				Limit:  uint64(Limit),
			}

			up, err := sdk.GetThingsByGroup(args[0], meta, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "thing <thing_id> <user_token>",
		Short: "Group by thing",
		Long:  `View group by specified thing`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.GetGroupByThing(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "profiles <group_id> <user_token>",
		Short: "Profiles by group",
		Long:  `Lists all profiles of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			meta := mfxsdk.PageMetadata{
				Offset: uint64(Offset),
				Limit:  uint64(Limit),
			}

			up, err := sdk.GetProfilesByGroup(args[0], meta, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "profile <profile_id> <user_token>",
		Short: "Group by profile",
		Long:  `View group by specified profile`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.GetGroupByProfile(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
}

// NewGroupsCmd returns users command.
func NewGroupsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "groups [create | get | delete | things | thing | profiles | profile]",
		Short: "Groups management",
		Long:  `Groups management: create, update, remove; get lists of: all groups, groups by org, things by group, profiles by group; get group by thing and by profile`,
	}

	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}

	return &cmd
}
