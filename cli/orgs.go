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
		Use:   "get [all | <org_id>] <user_token>",
		Short: "Get org",
		Long: `Get all orgs or org by id.
		all - lists all orgs
		<org_id> - shows org with provided org ID`,
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
	{
		Use:   "member <org_id> <member_id> <user_token>",
		Short: "View member",
		Long:  `View member by specified org`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.ViewMember(args[0], args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "assign <JSON_members> <org_id> <user_token>",
		Short: "Assign a member to org",
		Long:  `Assign a member to org`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var members []mfxsdk.OrgMember
			if err := json.Unmarshal([]byte(args[0]), &members); err != nil {
				logError(err)
				return
			}

			if err := sdk.AssignMembers(members, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "unassign <JSON_members> <org_id> <user_token>",
		Short: "Unassign a member from org",
		Long:  `Unassign a member from org`,
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

			if err := sdk.UnassignMembers(args[2], args[1], ids...); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "update-members <JSON_members> <org_id> <user_token>",
		Short: "Update members",
		Long:  `Update members by org`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var members []mfxsdk.OrgMember
			if err := json.Unmarshal([]byte(args[0]), &members); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateMembers(members, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "members <org_id> <user_token>",
		Short: "Members by org",
		Long:  `Lists members by org.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.ListMembersByOrg(args[0], args[1], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "memberships <member_id> <user_token>",
		Short: "Orgs by member",
		Long:  `Lists orgs by member.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.ListOrgsByMember(args[0], args[1], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
}

// NewOrgsCmd returns users command.
func NewOrgsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "orgs [create | get | delete | update | member | membership | assign | unassign | update-members | members | memberships]",
		Short: "Orgs management",
		Long:  `Orgs management: create, get, update or delete Org, get list of members by org and list of orgs by member, assigns members to org"`,
	}

	for i := range cmdOrgs {
		cmd.AddCommand(&cmdOrgs[i])
	}

	return &cmd
}
