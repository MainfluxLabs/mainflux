// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"

	"github.com/MainfluxLabs/mainflux/cli"
	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

const defURL string = "http://localhost"

func main() {
	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		AuthURL:         fmt.Sprintf("%s/svcauth", defURL),
		ThingsURL:       fmt.Sprintf("%s/svcthings", defURL),
		UsersURL:        fmt.Sprintf("%s/svcusers", defURL),
		WebhooksURL:     fmt.Sprintf("%s/svcwebhooks", defURL),
		ReaderURL:       fmt.Sprintf("%s/reader", defURL),
		HTTPAdapterURL:  fmt.Sprintf("%s/http", defURL),
		CertsURL:        defURL,
		MsgContentType:  sdk.ContentType(msgContentType),
		TLSVerification: false,
	}

	// Root
	var rootCmd = &cobra.Command{
		Use: "mainfluxlabs-cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			sdkConf.MsgContentType = sdk.ContentType(msgContentType)
			s := sdk.NewSDK(sdkConf)
			cli.SetSDK(s)
		},
	}

	// API commands
	healthCmd := cli.NewHealthCmd()
	usersCmd := cli.NewUsersCmd()
	thingsCmd := cli.NewThingsCmd()
	groupsCmd := cli.NewGroupsCmd()
	profilesCmd := cli.NewProfilesCmd()
	orgsCmd := cli.NewOrgsCmd()
	orgMembershipsCmd := cli.NewOrgMembershipsCmd()
	groupMembershipsCmd := cli.NewGroupMembershipsCmd()
	webhooksCmd := cli.NewWebhooksCmd()
	messagesCmd := cli.NewMessagesCmd()
	provisionCmd := cli.NewProvisionCmd()
	certsCmd := cli.NewCertsCmd()
	keysCmd := cli.NewKeysCmd()

	// Root Commands
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(groupsCmd)
	rootCmd.AddCommand(thingsCmd)
	rootCmd.AddCommand(profilesCmd)
	rootCmd.AddCommand(orgsCmd)
	rootCmd.AddCommand(orgMembershipsCmd)
	rootCmd.AddCommand(groupMembershipsCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(messagesCmd)
	rootCmd.AddCommand(provisionCmd)
	rootCmd.AddCommand(certsCmd)
	rootCmd.AddCommand(keysCmd)

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.AuthURL,
		"auth-url",
		"a",
		sdkConf.AuthURL,
		"Auth service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.CertsURL,
		"certs-url",
		"c",
		sdkConf.CertsURL,
		"Certs service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.ThingsURL,
		"things-url",
		"t",
		sdkConf.ThingsURL,
		"Things service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.UsersURL,
		"users-url",
		"u",
		sdkConf.UsersURL,
		"Users service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.WebhooksURL,
		"webhooks-url",
		"w",
		sdkConf.WebhooksURL,
		"Webhooks service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.HTTPAdapterURL,
		"http-url",
		"p",
		sdkConf.HTTPAdapterURL,
		"HTTP adapter URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&msgContentType,
		"content-type",
		"y",
		msgContentType,
		"Message content type",
	)

	rootCmd.PersistentFlags().BoolVarP(
		&sdkConf.TLSVerification,
		"insecure",
		"i",
		sdkConf.TLSVerification,
		"Do not check for TLS cert",
	)

	rootCmd.PersistentFlags().BoolVarP(
		&cli.RawOutput,
		"raw",
		"r",
		cli.RawOutput,
		"Enables raw output mode for easier parsing of output",
	)

	// Client and Profiles Flags
	rootCmd.PersistentFlags().UintVarP(
		&cli.Limit,
		"limit",
		"l",
		100,
		"Limit query parameter",
	)

	rootCmd.PersistentFlags().UintVarP(
		&cli.Offset,
		"offset",
		"o",
		0,
		"Offset query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Name,
		"name",
		"n",
		"",
		"Name query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Email,
		"email",
		"e",
		"",
		"User email query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Metadata,
		"metadata",
		"m",
		"",
		"Metadata query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Format,
		"format",
		"f",
		"",
		"Message format query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Subtopic,
		"subtopic",
		"s",
		"",
		"Subtopic query parameter",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.Publisher,
		"publisher",
		"",
		"Publisher ID query parameter",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.Protocol,
		"protocol",
		"",
		"Protocol query parameter",
	)

	rootCmd.PersistentFlags().Int64Var(
		&cli.From,
		"from",
		0,
		"From timestamp in milliseconds",
	)

	rootCmd.PersistentFlags().Int64Var(
		&cli.To,
		"to",
		0,
		"To timestamp in milliseconds",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.Dir,
		"dir",
		"",
		"Sort direction (asc/desc)",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.Filter,
		"filter",
		"",
		"Filter query parameter (JSON messages)",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.AggInterval,
		"agg-interval",
		"",
		"Aggregation interval (minute, hour, day, week, month, year)",
	)

	rootCmd.PersistentFlags().UintVar(
		&cli.AggValue,
		"agg-value",
		1,
		"Aggregation value",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.AggType,
		"agg-type",
		"",
		"Aggregation type (min, max, avg, count)",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.AggField,
		"agg-field",
		"",
		"Aggregation fields (comma-separated)",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.SenMLName,
		"senml-name",
		"",
		"SenML name filter",
	)

	rootCmd.PersistentFlags().Float64Var(
		&cli.SenMLValue,
		"senml-value",
		0,
		"SenML numeric value filter",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.Comparator,
		"comparator",
		"",
		"Comparison operator (eq, lt, le, gt, ge)",
	)

	rootCmd.PersistentFlags().BoolVar(
		&cli.BoolValue,
		"bool-value",
		false,
		"SenML boolean value filter",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.StringValue,
		"string-value",
		"",
		"SenML string value filter",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.DataValue,
		"data-value",
		"",
		"SenML data value filter",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.ConvertFormat,
		"convert",
		"json",
		"Export format (json/csv)",
	)

	rootCmd.PersistentFlags().StringVar(
		&cli.TimeFormat,
		"time-format",
		"",
		"Export time format",
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
