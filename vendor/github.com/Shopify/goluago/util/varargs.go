package util

import "github.com/Shopify/go-lua"

func PullVarargs(l *lua.State, startIndex int) ([]interface{}, error) {
	top := l.Top()
	if top < startIndex {
		return []interface{}{}, nil
	}

	varargs := make([]interface{}, top-startIndex+1)
	for i := startIndex; i <= top; i++ {
		var value interface{}
		var err error
		switch l.TypeOf(i) {
		case lua.TypeNil:
			value = nil
		case lua.TypeBoolean:
			value = l.ToBoolean(i)
		case lua.TypeLightUserData:
			value = nil // not supported by go-lua
		case lua.TypeNumber:
			value = lua.CheckNumber(l, i)
		case lua.TypeString:
			value = lua.CheckString(l, i)
		case lua.TypeTable:
			value, err = PullTable(l, i)
			if err != nil {
				return nil, err
			}
		case lua.TypeFunction:
			value = l.ToGoFunction(i)
		case lua.TypeUserData:
			value = l.ToUserData(i)
		case lua.TypeThread:
			value = l.ToThread(i)
		}
		varargs[i-startIndex] = value
	}
	return varargs, nil
}

func MustPullVarargs(l *lua.State, startIndex int) []interface{} {
	varargs, err := PullVarargs(l, startIndex)
	if err != nil {
		lua.Errorf(l, err.Error())
		panic("unreachable")
	}
	return varargs
}
