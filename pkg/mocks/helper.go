// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sort"
)

func SortItems[T any](order, dir string, items []T, getFields func(i int) (string, string)) []T {
	sort.SliceStable(items, sortByMeta(order, dir, getFields))
	return items
}

func sortByMeta(order, dir string, getFields func(i int) (string, string)) func(i, j int) bool {
	return func(i, j int) bool {
		nameI, idI := getFields(i)
		nameJ, idJ := getFields(j)

		switch order {
		case "name":
			if dir == "asc" {
				return nameI < nameJ
			}
			return nameI > nameJ
		case "id":
			if dir == "asc" {
				return idI < idJ
			}
			return idI > idJ
		default:
			return idI < idJ
		}
	}
}
