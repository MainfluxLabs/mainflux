// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdThings = []cobra.Command{
	{
		Use:   "create <JSON_thing> <group_id> <user_token>",
		Short: "Create thing",
		Long:  `Create new thing, generate his UUID and store it`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var thing mfxsdk.Thing
			if err := json.Unmarshal([]byte(args[0]), &thing); err != nil {
				logError(err)
				return
			}

			id, err := sdk.CreateThing(thing, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	{
		Use:   "get <all | by-profile | by-id> <id> <user_token>",
		Short: "Get things",
		Long: `Get all things, get things by profile or get thing by id.List of all things can be filtered by name or metadata
		all - lists all things
		by-profile - list things by profile based on defined <id>
		by-id - shows thing with provided <id>`,
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
				l, err := sdk.Things(args[1], pageMetadata)
				if err != nil {
					logError(err)
					return
				}

				logJSON(l)
				return
			case "by-profile":
				tip, err := sdk.ThingsByProfile(args[1], args[2], uint64(Offset), uint64(Limit))
				if err != nil {
					logError(err)
					return
				}

				logJSON(tip)
				return
			case "by-id":
				t, err := sdk.Thing(args[1], args[2])
				if err != nil {
					logError(err)
					return
				}

				logJSON(t)
				return
			default:
				logUsage(cmd.Use)
				return
			}
		},
	},
	{
		Use:   "metadata <thing_key>",
		Short: "Get thing metadata",
		Long:  "Get metadata about the thing identified by the given key",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			meta, err := sdk.MetadataByKey(args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(meta)
		},
	},
	{
		Use:   "delete <thing_id> <user_token>",
		Short: "Delete thing",
		Long:  `Removes thing from database`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.DeleteThing(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "identify <thing_key>",
		Short: "Identify thing",
		Long:  "Validates thing's key and returns its ID",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			i, err := sdk.IdentifyThing(args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(i)
		},
	},
	{
		Use:   "update <JSON_string> <thing_id> <user_token>",
		Short: "Update thing",
		Long:  `Update thing record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var thing mfxsdk.Thing
			if err := json.Unmarshal([]byte(args[0]), &thing); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateThing(thing, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewThingsCmd returns things command.
func NewThingsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "things [create | get | update | delete | identify | metadata]",
		Short: "Things management",
		Long:  `Things management: create, get, update, identify or delete Thing, get thing metadata, get a list of Things assigned to the specified Profile`,
	}

	for i := range cmdThings {
		cmd.AddCommand(&cmdThings[i])
	}

	return &cmd
}
