package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroupMemberships = []cobra.Command{
	{
		Use:   "create <JSON_memberships> <group_id> <user_token>",
		Short: "Create group memberships",
		Long:  `Create memberships for users by assigning a roles in the group`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			var memberships []mfxsdk.GroupMembership
			if err := json.Unmarshal([]byte(args[0]), &memberships); err != nil {
				logError(err)
				return
			}
			err := sdk.CreateGroupMemberships(memberships, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "get <group_id> <user_token>",
		Short: "Memberships by group",
		Long:  `Lists all memberships of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			meta := mfxsdk.PageMetadata{
				Offset: uint64(Offset),
				Limit:  uint64(Limit),
			}

			up, err := sdk.ListGroupMemberships(args[0], meta, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "update <JSON_memberships> <group_id> <user_token>",
		Short: "Update group memberships",
		Long:  `Update memberships by changing member roles`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var memberships []mfxsdk.GroupMembership
			if err := json.Unmarshal([]byte(args[0]), &memberships); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateGroupMemberships(memberships, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "delete <JSON_memberships> <group_id> <user_token>",
		Short: "Delete memberships from group",
		Long:  `Delete memberships from group.`,
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
			if err := sdk.RemoveGroupMemberships(ids, args[1], args[2]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
}

// NewGroupMembershipsCmd returns users command.
func NewGroupMembershipsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "group_memberships [create | get | update | delete]",
		Short: "Group memberships management",
		Long:  `Group memberships management: create, update, delete and list group memberships"`,
	}

	for i := range cmdGroupMemberships {
		cmd.AddCommand(&cmdGroupMemberships[i])
	}

	return &cmd
}
