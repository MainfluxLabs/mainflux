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
		AuthURL:         defURL,
		ThingsURL:       defURL,
		WebhooksURL:     defURL,
		UsersURL:        defURL,
		ReaderURL:       defURL,
		HTTPAdapterURL:  fmt.Sprintf("%s/http", defURL),
		BootstrapURL:    defURL,
		CertsURL:        defURL,
		MsgContentType:  sdk.ContentType(msgContentType),
		TLSVerification: false,
	}

	// Root
	var rootCmd = &cobra.Command{
		Use: "mainfluxlabs-cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cli.ParseConfig()

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
	channelsCmd := cli.NewChannelsCmd()
	webhooksCmd := cli.NewWebhooksCmd()
	orgsCmd := cli.NewOrgsCmd()
	groupRolesCmd := cli.NewGroupRolesCmd()
	messagesCmd := cli.NewMessagesCmd()
	provisionCmd := cli.NewProvisionCmd()
	certsCmd := cli.NewCertsCmd()
	keysCmd := cli.NewKeysCmd()

	// Root Commands
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(groupsCmd)
	rootCmd.AddCommand(thingsCmd)
	rootCmd.AddCommand(channelsCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(orgsCmd)
	rootCmd.AddCommand(groupRolesCmd)
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
		"s",
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
		&sdkConf.WebhooksURL,
		"webhooks-url",
		"w",
		sdkConf.WebhooksURL,
		"Webhooks service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.UsersURL,
		"users-url",
		"u",
		sdkConf.UsersURL,
		"Users service URL",
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

	rootCmd.PersistentFlags().StringVarP(
		&cli.ConfigPath,
		"config",
		"c",
		cli.ConfigPath,
		"Config path",
	)

	rootCmd.PersistentFlags().BoolVarP(
		&cli.RawOutput,
		"raw",
		"r",
		cli.RawOutput,
		"Enables raw output mode for easier parsing of output",
	)

	// Client and Channels Flags
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

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
