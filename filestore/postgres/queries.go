package postgres

import (
	"fmt"
	"strings"
)

func getFileNameQuery(name string) (string, string) {
	if name == "" {
		return "", ""
	}
	name = fmt.Sprintf(`%%%s%%`, strings.ToLower(name))
	return `LOWER(file_name) LIKE :file_name`, name
}

func getFileOrderQuery(order string) string {
	switch order {
	case "name":
		return "LOWER(file_name)"
	case "class":
		return "file_class"
	case "format":
		return "file_format"
	default:
		return "time"
	}
}

func getClassQuery(class string) string {
	if class == "" {
		return ""
	}
	return "file_class = :file_class"
}

func getFormatQuery(format string) string {
	if format == "" {
		return ""
	}
	return "file_format = :file_format"
}

func getThingQuery(thingID string) string {
	if thingID == "" {
		return ""
	}
	return "thing_id = :thing_id"
}

func getGroupQuery(groupID string) string {
	if groupID == "" {
		return ""
	}
	return "group_id = :group_id"
}
