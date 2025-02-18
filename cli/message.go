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
		Use:   "read <thing_key>",
		Short: "Read messages",
		Long:  `Reads all messages`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			pm := mfxsdk.PageMetadata{
				Offset:   uint64(Offset),
				Limit:    uint64(Limit),
				Format:   Format,
				Subtopic: Subtopic,
			}

			m, err := sdk.ReadMessages(pm, args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(m)
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
