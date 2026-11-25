package rules

import (
	"context"
	"fmt"
	"math"
	"strconv"
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

	// maxResultSize bounds the converted size of a returned value, counting every table entry
	// separately so that cyclic and repeatedly referenced tables are bounded as well.
	maxResultSize = 65_536
)

var (
	ErrScriptSize = errors.New("script size exceeds limit")

	errResultSize  = errors.New("script return value exceeds size limit")
	errResultKey   = errors.New("script return value contains an unsupported or duplicate table key")
	errResultValue = errors.New("script return value cannot be represented in JSON")
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
	RuleID     string
	ThingID    string
	Logs       []string
	StartedAt  time.Time
	FinishedAt time.Time
	Status     string
	// Human-readable string representing a runtime error during the execution of the lua script. May be an empty string in case of no error.
	Error string
	// Error value returned by lua.DoString
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
	// The ID of the Rule condition that triggered this execution
	ruleID string

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

	if env.instructionCount >= maxLuaInstructions {
		lua.Errorf(ls, "instruction count limit exceeded")
	}
}

// luaAPIFunc represents a Golang function exposed to the Lua scripting API.
type luaAPIFunc struct {
	// fun is called to obtain the actual Lua function. The return value of this lua.Function
	// represents the number of return values returned by it.
	fun func(env *luaEnv) lua.Function

	// The maximum number of invocations allowed per Lua script.
	maxInvocations uint

	// The function's identifier in the mfx Lua API namespace.
	identifier string
}

// Initializes and returns a new script environment associated with a specific Lua script and Mainflux message.
// The following is made available to the Lua environment as part of an "mfx" table in the global namespace:
//  1. The associated Message payload, subtopic, creation timestamp, and thing ID
//  2. API functions
func NewLuaEnv(service *rulesService, script *LuaScript, ruleID string, message *protomfx.Message, payload map[string]any, functions ...luaAPIFunc) (*luaEnv, error) {
	state := lua.NewState()

	env := &luaEnv{
		service: service,
		script:  script,
		ruleID:  ruleID,
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

	// dofile and loadfile read the service filesystem, and load compiles chunks outside the script.
	for _, name := range []string{"dofile", "loadfile", "load"} {
		state.PushNil()
		state.SetGlobal(name)
	}

	return env, nil
}

// Run the environment's associated Lua script and return its last return value.
func (env *luaEnv) execute() (ScriptRun, any, error) {
	id, err := env.service.idProvider.ID()
	if err != nil {
		return ScriptRun{}, nil, err
	}

	run := ScriptRun{
		ID:        id,
		ScriptID:  env.script.ID,
		ThingID:   env.message.ThingId,
		RuleID:    env.ruleID,
		StartedAt: time.Now(),
		Status:    ScriptRunStatusSuccess,
	}

	runErr := lua.DoString(env.ls, env.script.Script)

	run.FinishedAt = time.Now()
	run.Logs = env.logs
	run.err = runErr

	if runErr != nil {
		run.Error = runErr.Error()
		run.Status = ScriptRunStatusFail
		return run, nil, nil
	}

	if env.ls.Top() == 0 {
		return run, nil, nil
	}

	c := resultConverter{ls: env.ls, size: maxResultSize}
	value, err := c.value(-1)
	if err != nil {
		run.Error = err.Error()
		run.Status = ScriptRunStatusFail
		return run, nil, nil
	}

	return run, value, nil
}

// resultConverter converts a script return value to a Go value within the remaining size.
type resultConverter struct {
	ls   *lua.State
	size int
}

// take deducts n from the remaining size and reports whether the size limit is respected.
func (c *resultConverter) take(n int) bool {
	c.size -= n
	return c.size >= 0
}

// value converts the Lua value at idx to a Go value.
func (c *resultConverter) value(idx int) (any, error) {
	if !c.take(1) {
		return nil, errResultSize
	}

	switch c.ls.TypeOf(idx) {
	case lua.TypeNil:
		return nil, nil
	case lua.TypeBoolean:
		return c.ls.ToBoolean(idx), nil
	case lua.TypeNumber:
		n, _ := c.ls.ToNumber(idx)
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, errResultValue
		}
		return n, nil
	case lua.TypeString:
		s, _ := c.ls.ToString(idx)
		if !c.take(len(s)) {
			return nil, errResultSize
		}
		return s, nil
	case lua.TypeTable:
		return c.table(idx)
	default:
		return nil, errResultValue
	}
}

// table converts the Lua table at idx to an array when its keys are the integers 1 to n,
// and to an object otherwise. Errors leave the stack unbalanced because the Lua state is
// discarded after the conversion.
func (c *resultConverter) table(idx int) (any, error) {
	if !c.ls.CheckStack(2) {
		return nil, errResultSize
	}
	idx = c.ls.AbsIndex(idx)

	obj := map[string]any{}
	c.ls.PushNil()
	for c.ls.Next(idx) {
		var key string
		switch c.ls.TypeOf(-2) {
		case lua.TypeString:
			key, _ = c.ls.ToString(-2)
		case lua.TypeNumber:
			n, _ := c.ls.ToNumber(-2)
			key = strconv.FormatFloat(n, 'f', -1, 64)
		default:
			return nil, errResultKey
		}
		if _, ok := obj[key]; ok {
			return nil, errResultKey
		}
		if !c.take(len(key)) {
			return nil, errResultSize
		}

		value, err := c.value(-1)
		if err != nil {
			return nil, err
		}
		obj[key] = value
		c.ls.Pop(1)
	}

	n := c.ls.RawLength(idx)
	if n == 0 || n != len(obj) {
		return obj, nil
	}

	arr := make([]any, n)
	for i := range arr {
		value, ok := obj[strconv.Itoa(i+1)]
		if !ok {
			return obj, nil
		}
		arr[i] = value
	}

	return arr, nil
}

// runScriptCondition runs the script identified by scriptID as a fail-closed condition.
func (rs *rulesService) runScriptCondition(ctx context.Context, msg *protomfx.Message, payload map[string]any, ruleID, scriptID string) RuleRunCondition {
	result := RuleRunCondition{Type: ConditionTypeScript, ScriptID: scriptID}

	script, err := rs.rules.RetrieveScriptByID(ctx, scriptID)
	if err != nil {
		rs.logger.Error(fmt.Sprintf("retrieving script with id %s for condition evaluation failed with error: %v", scriptID, err))
		result.Error = err.Error()
		return result
	}

	env, err := NewLuaEnv(rs, &script, ruleID, msg, payload, luaLog)
	if err != nil {
		rs.logger.Error(fmt.Sprintf("creating lua environment for script with id %s failed with error: %v", scriptID, err))
		result.Error = err.Error()
		return result
	}

	run, value, err := env.execute()
	if err != nil {
		rs.logger.Error(fmt.Sprintf("executing script with id %s failed with error: %v", scriptID, err))
		result.Error = err.Error()
		return result
	}

	met, _ := value.(bool)
	result.Met = met
	result.Result = value
	result.Logs = run.Logs
	result.Error = run.Error

	// Runs of unmet conditions are skipped to avoid storing a run for every message a rule evaluates.
	if !met && run.Status != ScriptRunStatusFail {
		return result
	}

	if _, err := rs.rules.SaveScriptRuns(ctx, run); err != nil {
		rs.logger.Error(fmt.Sprintf("preserving script run to database failed with error: %v", err))
	}

	return result
}

// Create a table containing Mainflux Message data and push it to the Lua stack. The fields of the table are:
//   - "payload": a table containing the Message payload
//   - "subtopic": the subtopic the message was published to
//   - "created": message creation time (unix timestamp)
//   - "thing_id": ID of thing the message was published by
func pushMfxMessageTable(ls *lua.State, message *protomfx.Message, payload map[string]any) {
	msgTable := map[string]any{
		"subtopic": message.Subtopic,
		"created":  message.Created,
		"thing_id": message.ThingId,
		"payload":  payload,
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
