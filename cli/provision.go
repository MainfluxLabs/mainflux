// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/cobra"
)

const jsonExt = ".json"
const csvExt = ".csv"

var cmdProvision = []cobra.Command{
	{
		Use:   "things <things_file> <group_id> <user_token>",
		Short: "Provision things",
		Long:  `Bulk create things`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				logError(err)
				return
			}

			things, err := thingsFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}

			things, err = sdk.CreateThings(things, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(things)
		},
	},
	{
		Use:   "profiles <profiles_file> <group_id> <user_token>",
		Short: "Provision profiles",
		Long:  `Bulk create profiles`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			profiles, err := profilesFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}

			profiles, err = sdk.CreateProfiles(profiles, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(profiles)
		},
	},
	{
		Use:   "test",
		Short: "test",
		Long:  `Provisions test setup: one test user, three things and two profiles.`,
		Run: func(cmd *cobra.Command, args []string) {
			numThings := 3
			numProfs := 2

			things := []mfxsdk.Thing{}
			profiles := []mfxsdk.Profile{}

			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			un := fmt.Sprintf("%s@email.com", namesgenerator.GetRandomName(0))

			// Create test User
			user := mfxsdk.User{
				Email:    un,
				Password: "12345678",
			}
			if _, err := sdk.RegisterUser(user); err != nil {
				logError(err)
				return
			}

			ut, err := sdk.CreateToken(user)
			if err != nil {
				logError(err)
				return
			}

			// Create test Organization
			orgID, err := sdk.CreateOrg(mfxsdk.Org{Name: namesgenerator.GetRandomName(0)}, ut)
			if err != nil {
				logError(err)
				return
			}

			g := mfxsdk.Group{
				Name: "gr",
			}

			grID, err := sdk.CreateGroup(g, orgID, ut)
			if err != nil {
				logError(err)
				return
			}

			gr, err := sdk.Group(grID, ut)
			if err != nil {
				logError(err)
				return
			}

			// Create profiles
			for i := 0; i < numProfs; i++ {
				n := fmt.Sprintf("p%d", i)

				c := mfxsdk.Profile{
					Name:    n,
					GroupID: grID,
				}

				profiles = append(profiles, c)
			}
			profiles, err = sdk.CreateProfiles(profiles, grID, ut)
			if err != nil {
				logError(err)
				return
			}

			// Create things
			for i := 0; i < numThings; i++ {
				n := fmt.Sprintf("d%d", i)

				t := mfxsdk.Thing{
					Name:      n,
					GroupID:   grID,
					ProfileID: profiles[0].ID,
				}

				things = append(things, t)
			}
			things, err = sdk.CreateThings(things, grID, ut)
			if err != nil {
				logError(err)
				return
			}

			logJSON(user, ut, gr, things, profiles)
		},
	},
}

// NewProvisionCmd returns provision command.
func NewProvisionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "provision [things | profiles | test]",
		Short: "Provision things and profiles from a config file",
		Long:  `Provision things and profiles: use json or csv file to bulk provision things and profiles`,
	}

	for i := range cmdProvision {
		cmd.AddCommand(&cmdProvision[i])
	}

	return &cmd
}

func thingsFromFile(path string) ([]mfxsdk.Thing, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []mfxsdk.Thing{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []mfxsdk.Thing{}, err
	}
	defer file.Close()

	things := []mfxsdk.Thing{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mfxsdk.Thing{}, err
			}

			if len(l) < 1 {
				return []mfxsdk.Thing{}, errors.New("empty line found in file")
			}

			thing := mfxsdk.Thing{
				Name: l[0],
			}

			things = append(things, thing)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&things)
		if err != nil {
			return []mfxsdk.Thing{}, err
		}
	default:
		return []mfxsdk.Thing{}, err
	}

	return things, nil
}

func profilesFromFile(path string) ([]mfxsdk.Profile, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []mfxsdk.Profile{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []mfxsdk.Profile{}, err
	}
	defer file.Close()

	profiles := []mfxsdk.Profile{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mfxsdk.Profile{}, err
			}

			if len(l) < 1 {
				return []mfxsdk.Profile{}, errors.New("empty line found in file")
			}

			profile := mfxsdk.Profile{
				Name: l[0],
			}

			profiles = append(profiles, profile)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&profiles)
		if err != nil {
			return []mfxsdk.Profile{}, err
		}
	default:
		return []mfxsdk.Profile{}, err
	}

	return profiles, nil
}
