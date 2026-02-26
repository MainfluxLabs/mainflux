// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"strings"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/spf13/cobra"
)

func buildJSONPageMetadata() mfxsdk.JSONPageMetadata {
	var aggFields []string
	if AggField != "" {
		aggFields = strings.Split(AggField, ",")
	}

	return mfxsdk.JSONPageMetadata{
		Offset:      uint64(Offset),
		Limit:       uint64(Limit),
		Subtopic:    Subtopic,
		Publisher:   Publisher,
		Protocol:    Protocol,
		From:        From,
		To:          To,
		Filter:      Filter,
		AggInterval: AggInterval,
		AggValue:    uint64(AggValue),
		AggType:     AggType,
		AggFields:   aggFields,
		Dir:         Dir,
	}
}

func buildSenMLPageMetadata() mfxsdk.SenMLPageMetadata {
	var aggFields []string
	if AggField != "" {
		aggFields = strings.Split(AggField, ",")
	}

	return mfxsdk.SenMLPageMetadata{
		Offset:      uint64(Offset),
		Limit:       uint64(Limit),
		Subtopic:    Subtopic,
		Publisher:   Publisher,
		Protocol:    Protocol,
		Name:        SenMLName,
		Value:       SenMLValue,
		Comparator:  Comparator,
		BoolValue:   BoolValue,
		StringValue: StringValue,
		DataValue:   DataValue,
		From:        From,
		To:          To,
		AggInterval: AggInterval,
		AggValue:    uint64(AggValue),
		AggType:     AggType,
		AggFields:   aggFields,
		Dir:         Dir,
	}
}

var cmdMessages = []cobra.Command{
	{
		Use:   "send [subtopic] <JSON_string> <key_type> <thing_key>",
		Short: "Send messages",
		Long:  `Sends message via HTTP adapter`,
		Run: func(cmd *cobra.Command, args []string) {
			switch len(args) {
			case 3:
				key := things.ThingKey{Type: args[1], Value: args[2]}
				if err := sdk.SendMessage("", args[0], key); err != nil {
					logError(err)
					return
				}
			case 4:
				key := things.ThingKey{Type: args[2], Value: args[3]}
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
		Long:  `Reads all messages (legacy command, use 'read json' or 'read senml' for full filtering)`,
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

var cmdReadJSON = &cobra.Command{
	Use:   "json [<key_type> <thing_key>] <token>",
	Short: "Read JSON messages",
	Long:  `Reads JSON messages with full filtering support. Use flags for advanced filtering.`,
	Run: func(cmd *cobra.Command, args []string) {
		pm := buildJSONPageMetadata()
		switch len(args) {
		case 1:
			m, err := sdk.ListJSONMessages(pm, args[0], things.ThingKey{})
			if err != nil {
				logError(err)
				return
			}

			logJSON(m)
		case 3:
			key := things.ThingKey{Type: args[0], Value: args[1]}
			m, err := sdk.ListJSONMessages(pm, args[2], key)
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
}

var cmdReadSenML = &cobra.Command{
	Use:   "senml [<key_type> <thing_key>] <token>",
	Short: "Read SenML messages",
	Long:  `Reads SenML messages with full filtering support. Use flags for advanced filtering.`,
	Run: func(cmd *cobra.Command, args []string) {
		pm := buildSenMLPageMetadata()
		switch len(args) {
		case 1:
			m, err := sdk.ListSenMLMessages(pm, args[0], things.ThingKey{})
			if err != nil {
				logError(err)
				return
			}

			logJSON(m)
		case 3:
			key := things.ThingKey{Type: args[0], Value: args[1]}
			m, err := sdk.ListSenMLMessages(pm, args[2], key)
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
}

var cmdDeleteMessages = []cobra.Command{
	{
		Use:   "json <publisher_id> <token>",
		Short: "Delete JSON messages by publisher",
		Long:  `Deletes JSON messages for the specified publisher. Use --from and --to flags for time range.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			pm := buildJSONPageMetadata()
			if err := sdk.DeleteJSONMessages(args[0], args[1], pm); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "senml <publisher_id> <token>",
		Short: "Delete SenML messages by publisher",
		Long:  `Deletes SenML messages for the specified publisher. Use --from and --to flags for time range.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			pm := buildSenMLPageMetadata()
			if err := sdk.DeleteSenMLMessages(args[0], args[1], pm); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "all-json <token>",
		Short: "Delete all JSON messages (admin)",
		Long:  `Deletes all JSON messages. Requires admin privileges. Use --from and --to flags for time range.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			pm := buildJSONPageMetadata()
			if err := sdk.DeleteAllJSONMessages(args[0], pm); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "all-senml <token>",
		Short: "Delete all SenML messages (admin)",
		Long:  `Deletes all SenML messages. Requires admin privileges. Use --from and --to flags for time range.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			pm := buildSenMLPageMetadata()
			if err := sdk.DeleteAllSenMLMessages(args[0], pm); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

var cmdExportMessages = []cobra.Command{
	{
		Use:   "json <token>",
		Short: "Export JSON messages",
		Long:  `Exports JSON messages. Use --convert (json/csv), --time-format, and --publisher flags.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			pm := buildJSONPageMetadata()
			data, err := sdk.ExportJSONMessages(args[0], pm, ConvertFormat, TimeFormat)
			if err != nil {
				logError(err)
				return
			}

			cmd.OutOrStdout().Write(data)
		},
	},
	{
		Use:   "senml <token>",
		Short: "Export SenML messages",
		Long:  `Exports SenML messages. Use --convert (json/csv), --time-format, and --publisher flags.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			pm := buildSenMLPageMetadata()
			data, err := sdk.ExportSenMLMessages(args[0], pm, ConvertFormat, TimeFormat)
			if err != nil {
				logError(err)
				return
			}

			cmd.OutOrStdout().Write(data)
		},
	},
}

var cmdBackup = &cobra.Command{
	Use:   "backup <token>",
	Short: "Backup all messages (admin)",
	Long:  `Backs up all JSON and SenML messages. Requires admin privileges.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			logUsage(cmd.Use)
			return
		}

		m, err := sdk.BackupMessages(args[0])
		if err != nil {
			logError(err)
			return
		}

		logJSON(m)
	},
}

var cmdRestore = &cobra.Command{
	Use:   "restore <file_path> <token>",
	Short: "Restore messages from backup (admin)",
	Long:  `Restores JSON and SenML messages from a backup file. Requires admin privileges.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			logUsage(cmd.Use)
			return
		}

		data, err := os.ReadFile(args[0])
		if err != nil {
			logError(err)
			return
		}

		if err := sdk.RestoreMessages(args[1], data); err != nil {
			logError(err)
			return
		}

		logOK()
	},
}

// NewMessagesCmd returns messages command.
func NewMessagesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "messages [send | read | delete | export | backup | restore]",
		Short: "Send, read, delete, export, backup, or restore messages",
		Long:  `Send or read messages using the http-adapter and the configured database reader`,
	}

	for i := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	readCmd := &cmdMessages[1]
	readCmd.AddCommand(cmdReadJSON)
	readCmd.AddCommand(cmdReadSenML)

	deleteCmd := &cobra.Command{
		Use:   "delete [json | senml | all-json | all-senml]",
		Short: "Delete messages",
		Long:  `Delete messages by publisher or delete all messages (admin)`,
	}

	for i := range cmdDeleteMessages {
		deleteCmd.AddCommand(&cmdDeleteMessages[i])
	}
	cmd.AddCommand(deleteCmd)

	exportCmd := &cobra.Command{
		Use:   "export [json | senml]",
		Short: "Export messages",
		Long:  `Export messages to JSON or CSV format`,
	}

	for i := range cmdExportMessages {
		exportCmd.AddCommand(&cmdExportMessages[i])
	}

	cmd.AddCommand(exportCmd)
	cmd.AddCommand(cmdBackup)
	cmd.AddCommand(cmdRestore)

	return &cmd
}
