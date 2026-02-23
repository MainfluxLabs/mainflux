package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/Shopify/go-lua"
	luautil "github.com/Shopify/goluago/util"
)

const (
	luaAPIRootTableName       = "mfx"
	maxLuaInstructions        = 1_000_000
	debugHookInstructionCount = 10_000

	maxLogLineLength = 2_048
	maxLogLineCount  = 256
)

var (
	ErrScriptSize = errors.New("script size exceedes limit")
)

// LuaScript represents a specific Lua script.
type LuaScript struct {
	ID      string
	GroupID string
	// Lua script content
	Script      string
	Name        string
	Description string
}

type LuaScriptsPage struct {
	Total   uint64
	Scripts []LuaScript
}

// ScriptRun represents a specific run of a certain Lua script
type ScriptRun struct {
	ID         string
	ScriptID   string
	ThingID    string
	Logs       []string
	StartedAt  time.Time
	FinishedAt time.Time
	Status     string
	// Human-readable string representing a runtime error during the execution of the lua script. May be an empty string in case of no error.
	Error string
	// Error value retruned by lua.DoString
	err error
}

type ScriptRunsPage struct {
	Total uint64
	Runs  []ScriptRun
}

const (
	ScriptRunStatusSuccess = "success"
	ScriptRunStatusFail    = "fail"
)

// luaEnv represents an isolated environment for executing a Lua script.
type luaEnv struct {
	service *rulesService
	ls      *lua.State

	// The Lua script associated with the environment
	script *LuaScript

	message protomfx.Message
	payload map[string]any

	// The total number of Lua VM instructions executed in the associated Lua State
	instructionCount uint

	// Log messages produced by the script.
	logs []string
}

// Exposes Golang functions to the Lua scripting API under the mfx table namespace.
func (env *luaEnv) bindLuaAPIFuncs(funcs ...luaAPIFunc) {
	env.ls.Global(luaAPIRootTableName)
	defer env.ls.Pop(1)

	for _, apiFunc := range funcs {
		luaFunc := apiFunc.fun(env)
		luaFunc = toInvocationLimitedLuaFunc(luaFunc, apiFunc.maxInvocations)

		env.ls.PushGoFunction(luaFunc)
		env.ls.SetField(-2, apiFunc.identifier)
	}
}

func (env *luaEnv) debugHook(ls *lua.State, record lua.Debug) {
	env.instructionCount += debugHookInstructionCount

	if env.instructionCount > maxLuaInstructions {
		lua.Errorf(ls, "instruction count limit exceeded")
	}
}

// luaAPIFunc represents a Golang function exposed to the Lua scripting API.
type luaAPIFunc struct {
	// fun is called to obtain the actual Lua function
	fun func(env *luaEnv) lua.Function

	// The maximum number of invocations allowed per Lua script.
	maxInvocations uint

	// The function's identifier in the mfx Lua API namespace.
	identifier string
}

// Initializes and returns a new script environment associated with a specific Lua script and Mainflux message.
// The following is made available to the Lua environment as part of an "mfx" table in the global namespace:
//  1. The associated Message payload, subtopic, creation timestamp, and publisher ID
//  2. API functions
func NewLuaEnv(service *rulesService, script *LuaScript, message *protomfx.Message, payload map[string]any, functions ...luaAPIFunc) (*luaEnv, error) {
	state := lua.NewState()

	env := &luaEnv{
		service: service,
		script:  script,
		message: *message,
		payload: payload,
		ls:      state,
		logs:    make([]string, 0, 16),
	}

	lua.SetDebugHook(state, env.debugHook, lua.MaskCount, debugHookInstructionCount)

	// Expose basic Lua standard libraries: base, math, string, table
	lua.BaseOpen(state)
	state.Pop(1)

	lua.MathOpen(state)
	state.SetGlobal("math")

	lua.StringOpen(state)
	state.SetGlobal("string")

	lua.TableOpen(state)
	state.SetGlobal("table")

	// Primary "mfx" table - holds all exposed data (message, etc...) and scripting API functions
	state.NewTable()

	// Expose mfx Message and associated attributes
	pushMfxMessageTable(state, message, payload)
	state.SetField(-2, "message")

	state.SetGlobal(luaAPIRootTableName)

	// Bind all passed API functions
	env.bindLuaAPIFuncs(functions...)

	// Unset print() from base library
	state.PushNil()
	state.SetGlobal("print")

	return env, nil
}

// Run the environment's associated Lua script.
func (env *luaEnv) execute() (ScriptRun, error) {
	id, err := env.service.idProvider.ID()
	if err != nil {
		return ScriptRun{}, err
	}

	run := ScriptRun{
		ID:        id,
		ScriptID:  env.script.ID,
		ThingID:   env.message.Publisher,
		StartedAt: time.Now(),
		Status:    ScriptRunStatusSuccess,
	}

	err = lua.DoString(env.ls, env.script.Script)

	run.FinishedAt = time.Now()
	run.Logs = env.logs
	run.err = err

	if run.err != nil {
		run.Error = run.err.Error()
		run.Status = ScriptRunStatusFail
	}

	return run, nil
}

// Create a table containing Mainflux Message data and push it to the Lua stack. The fields of the table are:
//   - "payload": a table containing the Message payload
//   - "subtopic": the subtopic the message was published to
//   - "created": message creation time (unix timestamp)
//   - "publisher_id": ID of thing the message was published by
func pushMfxMessageTable(ls *lua.State, message *protomfx.Message, payload map[string]any) {
	msgTable := map[string]any{
		"subtopic":     message.Subtopic,
		"created":      message.Created,
		"publisher_id": message.Publisher,
		"payload":      payload,
	}

	luautil.DeepPush(ls, msgTable)
}

// Returns luaFunc decorated with a function that ensures it cannot be called more than `maxInvocations` times.
func toInvocationLimitedLuaFunc(luaFunc lua.Function, maxInvocations uint) lua.Function {
	if maxInvocations == 0 {
		return luaFunc
	}

	currentInvocations := uint(0)

	limitedFunc := func(ls *lua.State) int {
		if currentInvocations >= maxInvocations {
			ls.PushBoolean(false)
			ls.PushString("invocation limit exceeded")
			return 2
		}

		retCount := luaFunc(ls)
		currentInvocations += 1

		return retCount
	}

	return limitedFunc
}

// For each passed Lua script, create a new Lua environment and execute the associated script which processes the `msg` Mainflux message.
// msg.Payload is ignored. parsedPayload represents the entire parsed payload of the associated message.
// For each of the passed Lua scripts:
// - If parsedPayload represents a top-level JSON object, it is passed to the Lua script environment in its entirety.
// - If parsedPayload represents a top-level JSON array, a separate Lua script environment is created for each of its children (which must be JSON objects).
func (rs *rulesService) processLuaScripts(ctx context.Context, msg *protomfx.Message, parsedPayload any, scripts ...LuaScript) {
	var payloads []map[string]any

	switch payload := parsedPayload.(type) {
	case map[string]any:
		payloads = append(payloads, payload)
	case []any:
		for _, subPayload := range payload {
			subObjPayload, ok := subPayload.(map[string]any)
			if !ok {
				rs.logger.Error("malformed payload array")
				continue
			}

			payloads = append(payloads, subObjPayload)
		}
	}

	for _, script := range scripts {
		for _, subPayload := range payloads {
			env, err := NewLuaEnv(rs, &script, msg, subPayload, luaAPISetStandard...)
			if err != nil {
				rs.logger.Error(fmt.Sprintf("creating lua environment for script with id %s failed with error: %v", script.ID, err))
				continue
			}

			run, err := env.execute()
			if err != nil {
				rs.logger.Info(fmt.Sprintf("attempting to execute script with id %s failed with error %v", env.script.ID, err))
			}

			if _, err := rs.rules.SaveScriptRuns(ctx, run); err != nil {
				rs.logger.Error(fmt.Sprintf("preserving script run to database failed with error: %v", err))
			}
		}
	}
}
