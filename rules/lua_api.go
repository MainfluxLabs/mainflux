package rules

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/Shopify/go-lua"
	luautil "github.com/Shopify/goluago/util"
)

var luaAPISetStandard = []luaAPIFunc{luaSMTPNotify, luaAlarmCreate, luaLog, luaReaderListMessages}

// Trigger a registered SMTP notifier by ID.
// Lua signature:
// mfx.smtp_notify(smtp_notifier_id) (bool, msg)
// On success it returns true, nil. On failure, it returns (false, <error_message>)
var luaSMTPNotify = luaAPIFunc{
	fun: func(env *luaEnv) lua.Function {
		return func(ls *lua.State) int {
			notifierID, ok := ls.ToString(1)
			if !ok {
				ls.PushBoolean(false)
				return 1
			}

			// Marshal current payload, create new protomfx Message, and publish it to the SMTP NATS subject
			encodedPayload, err := json.Marshal(env.payload)
			if err != nil {
				ls.PushBoolean(false)
				ls.PushString(err.Error())
				return 2
			}

			msg := env.message
			msg.Payload = encodedPayload
			subject := fmt.Sprintf("%s.%s", subjectSMTP, notifierID)

			if err := env.service.pub.Publish(subject, msg); err != nil {
				ls.PushBoolean(false)
				ls.PushString(err.Error())
				return 2
			}

			ls.PushBoolean(true)
			return 1
		}
	},
	identifier:     "smtp_notify",
	maxInvocations: 2,
}

// Create an Alarm corresponding to the ID of the currently executing Lua script
// Lua signature:
// mfx.create_alarm()
// On success it returns true, nil. On failure, it returns (false, <error_message>)
var luaAlarmCreate = luaAPIFunc{
	fun: func(env *luaEnv) lua.Function {
		return func(ls *lua.State) int {
			subject := fmt.Sprintf("%s.%s.%s", subjectAlarms, alarms.AlarmOriginScript, env.script.ID)
			if err := env.service.pub.PublishAlarm(subject, protomfx.Alarm{
				ThingId:  env.message.Publisher,
				Subtopic: env.message.Subtopic,
				Protocol: env.message.Protocol,
				Created:  env.message.Created,
				RuleId:   env.script.ID,
			}); err != nil {
				ls.PushBoolean(false)
				ls.PushString(err.Error())
				return 2
			}

			ls.PushBoolean(true)
			return 1
		}
	},
	identifier:     "create_alarm",
	maxInvocations: 1,
}

// Log a string message to the environment's log buffer.
// Lua signature:
// mfx.log(<message>)
// Returns true on success, and (false, <error_message>) on failure, usually due to exceeded log size limits per environment.
var luaLog = luaAPIFunc{
	fun: func(env *luaEnv) lua.Function {
		return func(ls *lua.State) int {
			if len(env.logs) > maxLogLineCount {
				ls.PushBoolean(false)
				ls.PushString("log count exceeded")
				return 2
			}

			message, ok := ls.ToString(1)
			if !ok {
				ls.PushBoolean(false)
				return 1
			}

			if len(message) > maxLogLineLength {
				ls.PushBoolean(false)
				ls.PushString("log message exceeds maximum length")
				return 2
			}

			env.logs = append(env.logs, message)

			ls.PushBoolean(true)
			return 1
		}
	},
	identifier: "log",
}

// An interface to readers.ListJSONMessages and readers.ListSenMLMessages.
// Lua signature:
//
//		    mfx.list_messages(<message_king>, <thing_key>, [<page_metadata>])
//		        Where <message_kind> is one of "senml" or "json", <thing_key> is a Lua table of the following structure: { type = "external"|"internal", value = <key_value> },
//			    and <page_metadata> is an optional Lua table representing pagination metadata and filters (see the definitions of domain.SenMLPageMetadata and readers.JSONPageMetadata).
//
//	            <thing_key> may also be nil, in which case messages from the thing associated with the currently-executing scripts are fetched.
//
//			    On success, it returns (<messages>, <total_messages>). On failure, it returns (nil, 0, <error_message).
var luaReaderListMessages = luaAPIFunc{
	fun: func(env *luaEnv) lua.Function {
		return func(ls *lua.State) int {
			messageKind := lua.CheckString(ls, 1)

			// Parse second arg into a domain.ThingKey
			var thingKey domain.ThingKey

			switch ls.TypeOf(2) {
			case lua.TypeTable:
				ls.Field(2, "type")
				thingKey.Type = lua.CheckString(ls, -1)
				ls.Field(2, "value")
				thingKey.Value = lua.CheckString(ls, -1)
				ls.Pop(2)
			case lua.TypeNil:
				// Explicit nil passed: obtain thing key of publisher associated with the currently-executing script
				key, err := env.service.things.GetKeyByThingID(context.Background(), env.message.Publisher)
				if err != nil {
					ls.PushNil()
					ls.PushInteger(0)
					ls.PushString(err.Error())
					return 2
				}

				thingKey = key
			default:
				// TypeNone (no argument passed), or some other type
				lua.ArgumentError(ls, 2, "expected table (thing key) or nil")
				panic("unreachable")
			}

			if err := apiutil.ValidateThingKey(thingKey); err != nil {
				ls.PushNil()
				ls.PushInteger(0)
				ls.PushString(err.Error())
				return 2
			}

			switch messageKind {
			case "json":
				var pm domain.JSONPageMetadata

				if ls.IsTable(3) {
					pm = luaTableToJSONPageMetadata(ls, 3)
				}

				page, err := env.service.readers.ListJSONMessages(context.Background(), thingKey, pm)
				if err != nil {
					ls.PushNil()
					ls.PushNumber(0)
					ls.PushString(err.Error())

					return 3
				}

				// Return ({messages}, total)
				// Messages are returned as a Lua array (table whose keys are integers) of tables representing JSON messages
				luautil.DeepPush(ls, page.Messages)
				ls.PushInteger(int(page.Total))

				return 2
			case "senml":
				var pm domain.SenMLPageMetadata

				if ls.IsTable(3) {
					pm = luaTableToSenMLPageMetadata(ls, 3)
				}

				page, err := env.service.readers.ListSenMLMessages(context.Background(), thingKey, pm)
				if err != nil {
					ls.PushNil()
					ls.PushInteger(0)
					ls.PushString(err.Error())

					return 3
				}

				// Return ({messages}, total)
				// Messages are returned as a Lua array (table whose keys are integers) of tables representing SenML messages
				ls.NewTable()
				for idx, msg := range page.Messages {
					pushSenMLMessageToLua(ls, msg.(senml.Message))
					ls.RawSetInt(-2, idx+1)
				}
				ls.PushInteger(int(page.Total))

				return 2
			default:
				lua.ArgumentError(ls, 1, `expected "json" or "senml"`)
				panic("unreachable")
			}
		}
	},

	identifier: "list_messages",
}

// Helper that parses a Lua table at tblIdx into a domain.JSONPageMetadata. On error, namely if the value of a key
// in the table doesn't match that of the associated struct field, an error is thrown in the Lua environment,
// and the function doesn't return.
func luaTableToJSONPageMetadata(ls *lua.State, tableIdx int) domain.JSONPageMetadata {
	tableIdx = ls.AbsIndex(tableIdx)
	var pm domain.JSONPageMetadata

	for _, field := range []struct {
		name string
		dest *uint64
	}{
		{"offset", &pm.Offset},
		{"limit", &pm.Limit},
		{"agg_value", &pm.AggValue},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeNumber:
			n, _ := ls.ToNumber(-1)
			*field.dest = uint64(n)
		default:
			ls.Pop(1)
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a number")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	for _, field := range []struct {
		name string
		dest *int64
	}{
		{"from", &pm.From},
		{"to", &pm.To},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeNumber:
			n, _ := ls.ToNumber(-1)
			*field.dest = int64(n)
		default:
			ls.Pop(1)
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a number")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	for _, field := range []struct {
		name string
		dest *string
	}{
		{"subtopic", &pm.Subtopic},
		{"publisher", &pm.Publisher},
		{"protocol", &pm.Protocol},
		{"filter", &pm.Filter},
		{"agg_interval", &pm.AggInterval},
		{"agg_type", &pm.AggType},
		{"dir", &pm.Dir},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeString:
			*field.dest = lua.CheckString(ls, -1)
		default:
			ls.Pop(1)
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a string")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	ls.Field(tableIdx, "agg_fields")
	switch ls.TypeOf(-1) {
	case lua.TypeNone, lua.TypeNil:
	case lua.TypeTable:
		aggFieldsIdx := ls.AbsIndex(-1)
		ls.PushNil()
		for ls.Next(aggFieldsIdx) {
			switch ls.TypeOf(-1) {
			case lua.TypeString:
				pm.AggFields = append(pm.AggFields, lua.CheckString(ls, -1))
			default:
				ls.Pop(2)
				lua.ArgumentError(ls, tableIdx, "field 'agg_fields' must be a table of strings")
				panic("unreachable")
			}
			ls.Pop(1)
		}
	default:
		ls.Pop(1)
		lua.ArgumentError(ls, tableIdx, "field 'agg_fields' must be a table")
		panic("unreachable")
	}

	ls.Pop(1)

	return pm
}

// Helper that parses a Lua table at tblIdx into a domain.SenMLPageMetadata. On error, namely if the value of a key
// in the table doesn't match that of the associated struct field, an error is thrown in the Lua environment,
// and the function doesn't return.
func luaTableToSenMLPageMetadata(ls *lua.State, tableIdx int) domain.SenMLPageMetadata {
	tableIdx = ls.AbsIndex(tableIdx)
	var pm domain.SenMLPageMetadata

	for _, field := range []struct {
		name string
		dest *uint64
	}{
		{"offset", &pm.Offset},
		{"limit", &pm.Limit},
		{"agg_value", &pm.AggValue},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeNumber:
			n, _ := ls.ToNumber(-1)
			*field.dest = uint64(n)
		default:
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a number")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	for _, field := range []struct {
		name string
		dest *int64
	}{
		{"from", &pm.From},
		{"to", &pm.To},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeNumber:
			n, _ := ls.ToNumber(-1)
			*field.dest = int64(n)
		default:
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a number")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	for _, field := range []struct {
		name string
		dest *float64
	}{
		{"v", &pm.Value},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeNumber:
			n, _ := ls.ToNumber(-1)
			*field.dest = n
		default:
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a number")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	for _, field := range []struct {
		name string
		dest *string
	}{
		{"subtopic", &pm.Subtopic},
		{"publisher", &pm.Publisher},
		{"protocol", &pm.Protocol},
		{"name", &pm.Name},
		{"comparator", &pm.Comparator},
		{"vs", &pm.StringValue},
		{"vd", &pm.DataValue},
		{"agg_interval", &pm.AggInterval},
		{"agg_type", &pm.AggType},
		{"dir", &pm.Dir},
	} {
		ls.Field(tableIdx, field.name)
		switch ls.TypeOf(-1) {
		case lua.TypeNone, lua.TypeNil:
		case lua.TypeString:
			*field.dest = lua.CheckString(ls, -1)
		default:
			lua.ArgumentError(ls, tableIdx, "field '"+field.name+"' must be a string")
			panic("unreachable")
		}
		ls.Pop(1)
	}

	ls.Field(tableIdx, "vb")
	switch ls.TypeOf(-1) {
	case lua.TypeNone, lua.TypeNil:
	case lua.TypeBoolean:
		pm.BoolValue = ls.ToBoolean(-1)
	default:
		lua.ArgumentError(ls, tableIdx, "field 'vb' must be a boolean")
		panic("unreachable")
	}
	ls.Pop(1)

	ls.Field(tableIdx, "agg_fields")
	switch ls.TypeOf(-1) {
	case lua.TypeNone, lua.TypeNil:
	case lua.TypeTable:
		aggFieldsIdx := ls.AbsIndex(-1)
		ls.PushNil()
		for ls.Next(aggFieldsIdx) {
			switch ls.TypeOf(-1) {
			case lua.TypeString:
				pm.AggFields = append(pm.AggFields, lua.CheckString(ls, -1))
			default:
				ls.Pop(2)
				lua.ArgumentError(ls, tableIdx, "field 'agg_fields' must be a table of strings")
				panic("unreachable")
			}
			ls.Pop(1)
		}
	default:
		lua.ArgumentError(ls, tableIdx, "field 'agg_fields' must be a table")
		panic("unreachable")
	}
	ls.Pop(1)

	return pm
}

// Pushes a Lua table representing the `msg` SenML message to the top of the stack. Struct pointer types with nil values are omitted
// from the table.
func pushSenMLMessageToLua(ls *lua.State, msg senml.Message) {
	ls.NewTable()

	ls.PushString(msg.Subtopic)
	ls.SetField(-2, "subtopic")

	ls.PushString(msg.Publisher)
	ls.SetField(-2, "publisher")

	ls.PushString(msg.Protocol)
	ls.SetField(-2, "protocol")

	ls.PushString(msg.Name)
	ls.SetField(-2, "name")

	ls.PushString(msg.Unit)
	ls.SetField(-2, "unit")

	ls.PushInteger(int(msg.Time))
	ls.SetField(-2, "time")

	ls.PushNumber(msg.UpdateTime)
	ls.SetField(-2, "update_time")

	if msg.Value != nil {
		ls.PushNumber(*msg.Value)
		ls.SetField(-2, "value")
	}

	if msg.StringValue != nil {
		ls.PushString(*msg.StringValue)
		ls.SetField(-2, "string_value")
	}

	if msg.DataValue != nil {
		ls.PushString(*msg.DataValue)
		ls.SetField(-2, "data_value")
	}

	if msg.BoolValue != nil {
		ls.PushBoolean(*msg.BoolValue)
		ls.SetField(-2, "bool_value")
	}

	if msg.Sum != nil {
		ls.PushNumber(*msg.Sum)
		ls.SetField(-2, "sum")
	}
}
