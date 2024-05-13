// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

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
		Use:   "channels <channels_file> <group_id> <user_token>",
		Short: "Provision channels",
		Long:  `Bulk create channels`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			channels, err := channelsFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}

			channels, err = sdk.CreateChannels(channels, args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(channels)
		},
	},
	{
		Use:   "connect <connections_file> <user_token>",
		Short: "Provision connections",
		Long:  `Bulk connect things to channels`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			connIDs, err := connectionsFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}

			err = sdk.Connect(connIDs, args[1])
			if err != nil {
				logError(err)
				return
			}
		},
	},
	{
		Use:   "test",
		Short: "test",
		Long: `Provisions test setup: one test user, two things and two channels. \
						Connect both things to one of the channels, \
						and only on thing to other channel.`,
		Run: func(cmd *cobra.Command, args []string) {
			numThings := 3
			numChan := 2
			things := []mfxsdk.Thing{}
			channels := []mfxsdk.Channel{}
			orgID := "1"

			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			rand.Seed(time.Now().UnixNano())
			un := fmt.Sprintf("%s@email.com", namesgenerator.GetRandomName(0))
			// Create test user
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

			// Create things
			for i := 0; i < numThings; i++ {
				n := fmt.Sprintf("d%d", i)

				t := mfxsdk.Thing{
					Name:    n,
					GroupID: grID,
				}

				things = append(things, t)
			}
			things, err = sdk.CreateThings(things, grID, ut)
			if err != nil {
				logError(err)
				return
			}

			var thIDs []string
			for _, th := range things {
				thIDs = append(thIDs, th.ID)
			}

			// Create channels
			for i := 0; i < numChan; i++ {
				n := fmt.Sprintf("c%d", i)

				c := mfxsdk.Channel{
					Name:    n,
					GroupID: grID,
				}

				channels = append(channels, c)
			}
			channels, err = sdk.CreateChannels(channels, grID, ut)
			if err != nil {
				logError(err)
				return
			}

			var chIDs []string
			for _, ch := range channels {
				chIDs = append(chIDs, ch.ID)
			}

			conIDs := mfxsdk.ConnectionIDs{
				ChannelID: channels[0].ID,
				ThingIDs:  []string{things[0].ID, things[1].ID},
			}
			if err := sdk.Connect(conIDs, ut); err != nil {
				logError(err)
				return
			}

			conIDs = mfxsdk.ConnectionIDs{
				ChannelID: channels[1].ID,
				ThingIDs:  []string{things[2].ID},
			}
			if err := sdk.Connect(conIDs, ut); err != nil {
				logError(err)
				return
			}

			logJSON(user, ut, gr, things, channels)
		},
	},
}

// NewProvisionCmd returns provision command.
func NewProvisionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "provision [things | channels | connect | test]",
		Short: "Provision things and channels from a config file",
		Long:  `Provision things and channels: use json or csv file to bulk provision things and channels`,
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

func channelsFromFile(path string) ([]mfxsdk.Channel, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []mfxsdk.Channel{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []mfxsdk.Channel{}, err
	}
	defer file.Close()

	channels := []mfxsdk.Channel{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mfxsdk.Channel{}, err
			}

			if len(l) < 1 {
				return []mfxsdk.Channel{}, errors.New("empty line found in file")
			}

			channel := mfxsdk.Channel{
				Name: l[0],
			}

			channels = append(channels, channel)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&channels)
		if err != nil {
			return []mfxsdk.Channel{}, err
		}
	default:
		return []mfxsdk.Channel{}, err
	}

	return channels, nil
}

func connectionsFromFile(path string) (mfxsdk.ConnectionIDs, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return mfxsdk.ConnectionIDs{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return mfxsdk.ConnectionIDs{}, err
	}
	defer file.Close()

	connections := mfxsdk.ConnectionIDs{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return mfxsdk.ConnectionIDs{}, err
			}

			if len(l) < 1 {
				return mfxsdk.ConnectionIDs{}, errors.New("empty line found in file")
			}

			connections.ThingIDs = append(connections.ThingIDs, l[0])
			connections.ChannelID = l[1]
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&connections)
		if err != nil {
			return mfxsdk.ConnectionIDs{}, err
		}
	default:
		return mfxsdk.ConnectionIDs{}, err
	}

	return connections, nil
}
