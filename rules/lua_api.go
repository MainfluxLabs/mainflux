package rules

import (
	"github.com/Shopify/go-lua"
)

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
