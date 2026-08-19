package cli

import (
	"github.com/spf13/cobra"
)

// NewCertsCmd returns certificate command.
func NewCertsCmd() *cobra.Command {
	var keySize uint16
	var keyType string
	var ttl string

	issueCmd := cobra.Command{
		Use:   "issue <thing_id> <user_token> [--keysize=2048] [--keytype=rsa] [--ttl=8760h]",
		Short: "Issue certificate",
		Long:  `Issues new certificate for a thing`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			thingID := args[0]

			c, err := sdk.IssueCert(thingID, int(keySize), keyType, ttl, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(c)
		},
	}

	issueCmd.Flags().Uint16Var(&keySize, "keysize", 2048, "certificate key strength in bits: 2048, 4096 (RSA) or 224, 256, 384, 512 (EC)")
	issueCmd.Flags().StringVar(&keyType, "keytype", "rsa", "certificate key type: RSA or EC")
	issueCmd.Flags().StringVar(&ttl, "ttl", "8760h", "certificate time to live (e.g. 8760h, 30m, 10s, or integer for hours)")

	getCmd := cobra.Command{
		Use:   "get <serial> <user_token>",
		Short: "Get certificate",
		Long:  `Gets certificate by its serial number`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			c, err := sdk.ViewCert(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	}

	revokeCmd := cobra.Command{
		Use:   "revoke <serial> <user_token>",
		Short: "Revoke certificate",
		Long:  `Revokes certificate by its serial number`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.RevokeCert(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	}

	renewCmd := cobra.Command{
		Use:   "renew <serial> <user_token>",
		Short: "Renew certificate",
		Long:  `Renews certificate by its serial number`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			c, err := sdk.RenewCert(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	}

	serialsCmd := cobra.Command{
		Use:   "serials <thing_id> <user_token>",
		Short: "List certificate serials",
		Long:  `Lists certificate serial numbers for a thing`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			cp, err := sdk.ListSerials(args[0], uint64(Offset), uint64(Limit), args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(cp)
		},
	}

	cmd := cobra.Command{
		Use:   "certs [issue | get | revoke | renew | serials]",
		Short: "Certificates management",
		Long:  `Certificates management: create, view, revoke, renew, and list certificates for things"`,
	}

	cmdCerts := []cobra.Command{
		issueCmd,
		getCmd,
		revokeCmd,
		renewCmd,
		serialsCmd,
	}

	for i := range cmdCerts {
		cmd.AddCommand(&cmdCerts[i])
	}

	return &cmd
}
