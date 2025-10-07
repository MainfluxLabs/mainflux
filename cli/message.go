// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdMessages = []cobra.Command{
	{
		Use:   "send [subtopic] <JSON_string> <key_type> <thing_key>",
		Short: "Send messages",
		Long:  `Sends message`,
		Run: func(cmd *cobra.Command, args []string) {
			switch len(args) {
			case 3:
				key := apiutil.ThingKey{Type: args[1], Value: args[2]}
				if err := sdk.SendMessage("", args[0], key); err != nil {
					logError(err)
					return
				}
			case 4:
				key := apiutil.ThingKey{Type: args[2], Value: args[3]}
				if err := sdk.SendMessage(args[0], args[1], key); err != nil {
					logError(err)
					return
				}
			default:
				logUsage(cmd.Use)
				return
			}

			logOK()
		},
	},
	{
		Use:   "read [by-admin] <key_type> <auth_token>",
		Short: "Read messages",
		Long:  `Reads all messages`,
		Run: func(cmd *cobra.Command, args []string) {
			pm := mfxsdk.PageMetadata{
				Offset:   uint64(Offset),
				Limit:    uint64(Limit),
				Format:   Format,
				Subtopic: Subtopic,
			}
			switch len(args) {
			case 2:
				m, err := sdk.ReadMessages(true, pm, "", args[1])
				if err != nil {
					logError(err)
					return
				}

				logJSON(m)
			case 3:
				m, err := sdk.ReadMessages(false, pm, args[0], args[1])
				if err != nil {
					logError(err)
					return
				}

				logJSON(m)
			default:
				logUsage(cmd.Use)
				return
			}
		},
	},
}

// NewMessagesCmd returns messages command.
func NewMessagesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "messages [send | read]",
		Short: "Send or read messages",
		Long:  `Send or read messages using the http-adapter and the configured database reader`,
	}

	for i := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	return &cmd
}
