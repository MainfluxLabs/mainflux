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
	"strings"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/cobra"
)

const jsonExt = ".json"
const csvExt = ".csv"

const CSV_THINGS_FIELD_COUNT = 4

// These constants define the order of the CSV columns (fields) of records containing Things to be provisioned
const (
	CSV_THINGS_FIELD_ID_IDX = iota
	CSV_THINGS_FIELD_NAME_IDX
	CSV_THINGS_FIELD_GROUP_ID_IDX
	CSV_THINGS_FIELD_PROFILE_ID_IDX
)

const CSV_PROFILES_FIELD_COUNT = 12

// These constants define the order of the CSV columns (fields) of records containing Profiles to be provisioned
const (
	CSV_PROFILES_FIELD_ID_IDX = iota
	CSV_PROFILES_FIELD_NAME_IDX
	CSV_PROFILES_FIELD_GROUP_ID_IDX
	CSV_PROFILES_FIELD_CONFIG_CONTENTTYPE_IDX
	CSV_PROFILES_FIELD_CONFIG_WRITE_IDX
	CSV_PROFILES_FIELD_CONFIG_WEBHOOK_IDX
	CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_DATA_FILTERS_IDX
	CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_DATA_FIELD_IDX
	CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_TIME_FIELD_IDX
	CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_TIME_FORMAT_IDX
	CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_TIME_LOCATION_IDX
	CSV_PROFILES_FIELD_CONFIG_SMTP_ID_IDX
)

var cmdProvision = []cobra.Command{
	{
		Use:   "things <things_file> <group_id> <user_token>",
		Short: "Provision things",
		Long:  `Bulk create things`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
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
			if len(args) != 3 {
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

			gr, err := sdk.GetGroup(grID, ut)
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
		reader.FieldsPerRecord = -1

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mfxsdk.Thing{}, err
			}

			if len(record) < CSV_THINGS_FIELD_COUNT {
				return []mfxsdk.Thing{}, errors.New("malformed record in csv file")
			}

			thing := mfxsdk.Thing{
				Name:      record[CSV_THINGS_FIELD_NAME_IDX],
				ID:        record[CSV_THINGS_FIELD_ID_IDX],
				ProfileID: record[CSV_THINGS_FIELD_PROFILE_ID_IDX],
				GroupID:   record[CSV_THINGS_FIELD_GROUP_ID_IDX],
			}

			recordMetadata := record[CSV_THINGS_FIELD_COUNT:]

			// Thing record includes metadata variables
			if len(recordMetadata) > 0 {
				// Un-paired metadata fields present, abort
				if len(recordMetadata)%2 != 0 {
					return []mfxsdk.Thing{}, errors.New("malformed record in csv file")
				}

				thing.Metadata = make(map[string]any, len(recordMetadata)/2)

				// Consume all key-value metadata pairs from current Thing record and save them to map
				for i := 0; i < len(recordMetadata); i += 2 {
					thing.Metadata[recordMetadata[i]] = recordMetadata[i+1]
				}
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
		reader.FieldsPerRecord = -1

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mfxsdk.Profile{}, err
			}

			if len(record) < CSV_PROFILES_FIELD_COUNT {
				return []mfxsdk.Profile{}, errors.New("malformed record in csv file")
			}

			profile := mfxsdk.Profile{
				Name:    record[CSV_PROFILES_FIELD_NAME_IDX],
				ID:      record[CSV_PROFILES_FIELD_ID_IDX],
				GroupID: record[CSV_PROFILES_FIELD_GROUP_ID_IDX],
			}

			// Populate profile's config object
			profile.Config = make(map[string]any)
			profile.Config["content_type"] = record[CSV_PROFILES_FIELD_CONFIG_CONTENTTYPE_IDX]

			switch record[CSV_PROFILES_FIELD_CONFIG_WRITE_IDX] {
			case "true":
				profile.Config["write"] = true
			case "false":
				profile.Config["write"] = false
			default:
				return []mfxsdk.Profile{}, errors.New("malformed record in csv file")
			}

			switch record[CSV_PROFILES_FIELD_CONFIG_WEBHOOK_IDX] {
			case "true":
				profile.Config["webhook"] = true
			case "false":
				profile.Config["webhook"] = false
			default:
				return []mfxsdk.Profile{}, errors.New("malformed record in csv file")
			}

			profile.Config["smpt_id"] = record[CSV_PROFILES_FIELD_CONFIG_SMTP_ID_IDX]

			profile.Config["transformer"] = map[string]any{}
			transformer := profile.Config["transformer"].(map[string]any)

			transformer["data_field"] = record[CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_DATA_FIELD_IDX]
			transformer["time_field"] = record[CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_TIME_FIELD_IDX]
			transformer["time_location"] = record[CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_TIME_LOCATION_IDX]
			transformer["time_format"] = record[CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_TIME_FORMAT_IDX]
			transformer["data_filters"] = strings.Split(record[CSV_PROFILES_FIELD_CONFIG_TRANSFORMER_DATA_FILTERS_IDX], ",")

			recordMetadata := record[CSV_PROFILES_FIELD_COUNT:]

			// Profile record includes metadata variables
			if len(recordMetadata) > 0 {
				// Un-paired metadata fields present, abort
				if len(recordMetadata)%2 != 0 {
					return []mfxsdk.Profile{}, errors.New("malformed record in csv file")
				}

				profile.Metadata = make(map[string]any, len(recordMetadata)/2)

				// Consume all key-value metadata pairs from current Thing record and save them to map
				for i := 0; i < len(recordMetadata); i += 2 {
					profile.Metadata[recordMetadata[i]] = recordMetadata[i+1]
				}
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
