package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdOrgMembers = []cobra.Command{
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
		Use:   "get <all | member_id> <org_id> <user_token>",
		Short: "Get org member or list members",
		Long: `Get all org members or get a specific org member.
		all  - list all org members by provided org_id
		member_id - shows org member with provided <member_id> and <org_id>`,
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
				mbs, err := sdk.ListMembersByOrg(args[1], meta, args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(mbs)
				return
			}

			mb, err := sdk.GetMember(args[0], args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logJSON(mb)
		},
	},
	{
		Use:   "update <JSON_members> <org_id> <user_token>",
		Short: "Update org members",
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

			if err := sdk.UnassignMembers(ids, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewOrgMembersCmd returns users command.
func NewOrgMembersCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "org_members [assign | get | update | unassign]",
		Short: "Org members management",
		Long:  `Org members management: assign, unassign, get or update org members"`,
	}

	for i := range cmdOrgMembers {
		cmd.AddCommand(&cmdOrgMembers[i])
	}

	return &cmd
}
