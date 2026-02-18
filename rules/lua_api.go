package rules

import (
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/Shopify/go-lua"
)

var luaAPISetStandard = []luaAPIFunc{luaSMTPNotify, luaAlarmCreate, luaLog}

// Trigger a registered SMTP notifier by ID.
// Lua signature:
// mfx.smtp_notify(smtp_notifier_id) (bool, msg)
// On success it returns true, nil. On failure, it returns false, <error_message>
var luaSMTPNotify = luaAPIFunc{
	fun: func(env *luaEnv) lua.Function {
		return func(ls *lua.State) int {
			notifier_id, ok := ls.ToString(1)
			if !ok {
				ls.PushBoolean(false)
				return 1
			}

			// marshal current payload, create new protomfx Message, and publish message to ap
			encodedPayload, err := json.Marshal(env.payload)
			if err != nil {
				ls.PushBoolean(false)
				ls.PushString(err.Error())
				return 2
			}

			msg := env.message
			msg.Payload = encodedPayload
			msg.Subject = fmt.Sprintf("%s.%s", subjectSMTP, notifier_id)

			if err := env.service.pubsub.Publish(msg); err != nil {
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
// On success it returns true, nil. On failure, it returns false, <error_message>
var luaAlarmCreate = luaAPIFunc{
	fun: func(env *luaEnv) lua.Function {
		return func(ls *lua.State) int {
			encodedPayload, err := json.Marshal(env.payload)
			if err != nil {
				ls.PushBoolean(false)
				ls.PushString(err.Error())
				return 2
			}

			msg := env.message
			msg.Payload = encodedPayload
			msg.Subject = fmt.Sprintf("%s.%s.%s", subjectAlarms, alarms.AlarmOriginScript, env.script.ID)

			if err := env.service.pubsub.Publish(msg); err != nil {
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
// Returns true on success, and false, <error_message> on failure, usually due to exceeded log size limits per environment.
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
