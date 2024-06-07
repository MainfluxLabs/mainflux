// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdChannels = []cobra.Command{
	{
		Use:   "create <JSON_channel> <group_id> <user_token>",
		Short: "Create channel",
		Long:  `Creates new channel and generates it's UUID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var channel mfxsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logError(err)
				return
			}

			id, err := sdk.CreateChannel(channel, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	{
		Use:   "get [all | <channel_id>] <user_token>",
		Short: "Get channel",
		Long: `Get all channels or get channel by id. Channels can be filtered by name or metadata.
		all - lists all channels
		<channel_id> - shows thing with provided <channel_id>`,

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
				l, err := sdk.Channels(args[1], pageMetadata)
				if err != nil {
					logError(err)
					return
				}

				logJSON(l)
				return
			}
			c, err := sdk.Channel(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
	{
		Use:   "update <JSON_string> <channel_id> <user_token>",
		Short: "Update channel",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var channel mfxsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateChannel(channel, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "delete <channel_id> <user_token>",
		Short: "Delete channel",
		Long:  `Delete channel by ID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.DeleteChannel(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "connections <channel_id> <user_token>",
		Short: "Connections list",
		Long:  `List of Things connected to a Channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			cl, err := sdk.ThingsByChannel(args[1], args[0], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}

			logJSON(cl)
		},
	},
}

// NewChannelsCmd returns channels command.
func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels [create | get | update | delete | connections]",
		Short: "Channels management",
		Long:  `Channels management: create, get, update or delete Channel and get list of Things connected to a Channel`,
	}

	for i := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	return &cmd
}
