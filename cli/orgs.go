package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdOrgs = []cobra.Command{
	{
		Use:   "create <JSON_org> <user_token>",
		Short: "Create org",
		Long: `Creates new org:
		{
			"Name":<org_name>,
			"Description":<description>,
			"Metadata":<metadata>,
		}
		Name - is unique org name
		Metadata - JSON structured string`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			var org mfxsdk.Org
			if err := json.Unmarshal([]byte(args[0]), &org); err != nil {
				logError(err)
				return
			}

			err := sdk.CreateOrg(org, args[1])
			if err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "get <all | org_id> <user_token>",
		Short: "Get org",
		Long: `Get all orgs or org by id.
		all - lists all orgs
		org_id - shows org with provided <org_id>`,
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
				l, err := sdk.Orgs(meta, args[1])
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
			t, err := sdk.Org(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(t)
		},
	},
	{
		Use:   "delete <org_id> <user_token>",
		Short: "Delete org",
		Long:  `Delete org.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			if err := sdk.DeleteOrg(args[0], args[1]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "update <JSON_org> <org_id> <user_token>",
		Short: "Update org",
		Long:  `Update org record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var org mfxsdk.Org
			if err := json.Unmarshal([]byte(args[0]), &org); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateOrg(org, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewOrgsCmd returns users command.
func NewOrgsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "orgs [create | get | delete | update]",
		Short: "Orgs management",
		Long:  `Orgs management: create, get, update or delete org"`,
	}

	for i := range cmdOrgs {
		cmd.AddCommand(&cmdOrgs[i])
	}

	return &cmd
}
