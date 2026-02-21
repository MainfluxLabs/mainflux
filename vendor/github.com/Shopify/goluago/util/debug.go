package util

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"io"
)

// DumpFrame writes the currently visible frames of the Lua stack in a
// human-readable way.
func DumpFrame(l *lua.State, w io.Writer) (n int, err error) {
	n, err = fmt.Fprintf(w, "top=%d, 'real (pseudo): val':\n", l.Top())
	if err != nil {
		return
	}
	var m int
	for i, pseudo := 1, 0-l.Top(); i <= l.Top(); {
		m, err = fmt.Fprintf(w, "\t %d (%d): %#v\n", i, pseudo, l.ToValue(i))
		n += m
		if err != nil {
			return
		}

		i++
		pseudo++
	}
	return
}
