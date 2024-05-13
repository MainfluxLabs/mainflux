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
		Use:   "create <JSON_group> <org_id> <user_auth_token>",
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
			if len(args) != 2 {
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
		Use:   "get [all | <group_id>] <user_auth_token>",
		Short: "Get group",
		Long: `Get all users groups or group by id.
		all - lists all groups
		<group_id> - shows group with provided group ID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsage(cmd.Use)
				return
			}
			if args[0] == "all" {
				if len(args) > 2 {
					logUsage(cmd.Use)
					return
				}
				meta := mfxsdk.PageMetadata{
					Offset: uint64(Offset),
					Limit:  uint64(Limit),
				}
				l, err := sdk.Groups(meta, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if len(args) > 2 {
				logUsage(cmd.Use)
				return
			}
			t, err := sdk.Group(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(t)
		},
	},
	{
		Use:   "delete <group_id> <user_auth_token>",
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
		Use:   "things <group_id> <user_auth_token>",
		Short: "Group things list",
		Long:  `Lists all things of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.ListGroupThings(args[0], args[1], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "membership <thing_id> <user_auth_token>",
		Short: "Thing membership",
		Long:  `View thing group membership`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.ViewThingGroup(args[0], args[1], uint64(Offset), uint64(Limit))
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
		Use:   "groups [create | get | delete | assign | unassign | members | membership]",
		Short: "Groups management",
		Long:  `Groups management: create groups and assigns member to groups"`,
	}

	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}

	return &cmd
}
