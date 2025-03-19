// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdMessages = []cobra.Command{
	{
		Use:   "send [subtopic] <JSON_string> <thing_key>",
		Short: "Send messages",
		Long:  `Sends message`,
		Run: func(cmd *cobra.Command, args []string) {
			switch len(args) {
			case 2:
				if err := sdk.SendMessage("", args[0], args[1]); err != nil {
					logError(err)
					return
				}
			case 3:
				if err := sdk.SendMessage(args[0], args[1], args[2]); err != nil {
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
		Use:   "read [by-admin] <auth_token>",
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
			case 1:
				m, err := sdk.ReadMessages(pm, false, args[0])
				if err != nil {
					logError(err)
					return
				}

				logJSON(m)
			case 2:
				m, err := sdk.ReadMessages(pm, true, args[1])
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
