package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroupRoles = []cobra.Command{
	{
		Use:   "create <JSON_roles> <group_id> <user_token>",
		Short: "Create roles by group ",
		Long:  `Creates new roles by group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			var gm []mfxsdk.GroupMember
			if err := json.Unmarshal([]byte(args[0]), &gm); err != nil {
				logError(err)
				return
			}
			err := sdk.CreateRolesByGroup(gm, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "get <group_id> <user_token>",
		Short: "Roles by group",
		Long:  `Lists all roles of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			up, err := sdk.ListRolesByGroup(args[0], args[1], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "update <JSON_role> <group_id> <user_token>",
		Short: "Update roles by group",
		Long:  `Update group roles record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var roles []mfxsdk.GroupMember
			if err := json.Unmarshal([]byte(args[0]), &roles); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateRolesByGroup(roles, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "delete <JSON_roles> <group_id> <user_token>",
		Short: "Delete roles by group",
		Long:  `Delete roles by group.`,
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
			if err := sdk.RemoveRolesByGroup(ids, args[1], args[2]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
}

// NewGroupRolesCmd returns users command.
func NewGroupRolesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "group_roles [create | get | update | delete]",
		Short: "Group roles management",
		Long:  `Group roles management: create, update, remove and list group roles"`,
	}

	for i := range cmdGroupRoles {
		cmd.AddCommand(&cmdGroupRoles[i])
	}

	return &cmd
}
