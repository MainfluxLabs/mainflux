// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"fmt"
	"sort"

	"github.com/MainfluxLabs/mainflux/things"
)

// Since mocks will store data in map, and they need to resemble the real
// identifiers as much as possible, a key will be created as combination of
// owner and their own identifiers. This will allow searching either by
// prefix or suffix.
func key(owner string, id string) string {
	return fmt.Sprintf("%s-%s", owner, id)
}

func sortItems[T any](pm things.PageMetadata, items []T, getFields func(i int) (string, string)) []T {
	sort.SliceStable(items, sortByMeta(pm, getFields))
	return items
}


func sortByMeta(pm things.PageMetadata, getFields func(i int) (string, string)) func(i, j int) bool {
	return func(i, j int) bool {
		nameI, idI := getFields(i)
		nameJ, idJ := getFields(j)

		switch pm.Order {
		case "name":
			if pm.Dir == "asc" {
				return nameI < nameJ
			}
			return nameI > nameJ
		case "id":
			if pm.Dir == "asc" {
				return idI < idJ
			}
			return idI > idJ
		default:
			return idI < idJ
		}
	}
}
