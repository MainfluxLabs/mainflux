package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdOrgMemberships = []cobra.Command{
	{
		Use:   "create <JSON_memberships> <org_id> <user_token>",
		Short: "Create org membership",
		Long:  `Create membership for a specific user by assigning a role in org`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var memberships []mfxsdk.OrgMembership
			if err := json.Unmarshal([]byte(args[0]), &memberships); err != nil {
				logError(err)
				return
			}

			if err := sdk.CreateOrgMemberships(memberships, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "get <all | member_id> <org_id> <user_token>",
		Short: "Get org membership or list memberships",
		Long: `Get all org memberships or get a specific org membership.
		all  - list all org memberships by provided org_id
		member_id - shows org membership with provided <member_id> and <org_id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			meta := mfxsdk.PageMetadata{
				Offset: uint64(Offset),
				Limit:  uint64(Limit),
			}

			if args[0] == "all" {
				mbs, err := sdk.ListOrgMemberships(args[1], meta, args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(mbs)
				return
			}

			mb, err := sdk.GetOrgMembership(args[0], args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logJSON(mb)
		},
	},
	{
		Use:   "update <JSON_memberships> <org_id> <user_token>",
		Short: "Update org memberships",
		Long:  `Update membership by changing member role`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var memberships []mfxsdk.OrgMembership
			if err := json.Unmarshal([]byte(args[0]), &memberships); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateOrgMemberships(memberships, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "remove <JSON_memberships> <org_id> <user_token>",
		Short: "Remove a membership from org",
		Long:  `Remove a membership from org`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var ids []string
			if err := json.Unmarshal([]byte(args[0]), &ids); err != nil {
				logError(err)
				return
			}

			if err := sdk.RemoveOrgMemberships(ids, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewOrgMembershipsCmd returns users command.
func NewOrgMembershipsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "org_memberships [create | get | update | remove]",
		Short: "Org memberships management",
		Long:  `Org memberships management: create, remove, get or update org memberships"`,
	}

	for i := range cmdOrgMemberships {
		cmd.AddCommand(&cmdOrgMemberships[i])
	}

	return &cmd
}
