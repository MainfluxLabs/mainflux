package cli

import (
	"encoding/json"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroupMembers = []cobra.Command{
	{
		Use:   "create <JSON_members> <group_id> <user_token>",
		Short: "Create group members",
		Long:  `Creates group members.`,
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
			err := sdk.CreateGroupMembers(gm, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "get <group_id> <user_token>",
		Short: "Members by group",
		Long:  `Lists all members of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			meta := mfxsdk.PageMetadata{
				Offset: uint64(Offset),
				Limit:  uint64(Limit),
			}

			up, err := sdk.ListGroupMembers(args[0], meta, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "update <JSON_member> <group_id> <user_token>",
		Short: "Update group member",
		Long:  `Update group member record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var members []mfxsdk.GroupMember
			if err := json.Unmarshal([]byte(args[0]), &members); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateGroupMembers(members, args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "delete <JSON_members> <group_id> <user_token>",
		Short: "Delete members from group",
		Long:  `Delete members from group.`,
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
			if err := sdk.RemoveGroupMembers(ids, args[1], args[2]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
}

// NewGroupMembersCmd returns users command.
func NewGroupMembersCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "group_members [create | get | update | delete]",
		Short: "Group members management",
		Long:  `Group members management: create, update, remove and list group members"`,
	}

	for i := range cmdGroupMembers {
		cmd.AddCommand(&cmdGroupMembers[i])
	}

	return &cmd
}
