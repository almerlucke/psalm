package psalm

type Connection struct {
	Module Modular
	Index  int
}

func (c *Connection) OutputChannels() Channels {
	return c.Module.Output(c.Index).Channels
}

// ConnectionDesc for quick connecting multiple module/control outputs to inputs of module/control
type ConnectionDesc struct {
	Module   Modular
	OutIndex int
	InIndex  int
}

// ParseConnectionDescs parse args to []*ConnectionDesc, args need to be in the following order:
// - required module whose output is routed to input index
// - optional outIndex of module (default 0)
// - if outIndex is given follows an optional inIndex (otherwise index of count of IConn's parsed so far)
// - repeat...
func ParseConnectionDescs(args ...any) []*ConnectionDesc {
	var (
		conns      []*ConnectionDesc
		inIndex    int
		outIndex   int
		argIndex   int
		inIndexCnt int
		n          = len(args)
	)

	for argIndex < n {
		mod := args[argIndex].(Modular)
		argIndex++

		outIndex = 0
		inIndex = inIndexCnt

		if argIndex < n {
			if givenOutIndex, ok := args[argIndex].(int); ok {
				outIndex = givenOutIndex
				argIndex++
			}
		}

		if argIndex < n {
			if givenInIndex, ok := args[argIndex].(int); ok {
				inIndex = givenInIndex
				argIndex++
			}
		}

		conns = append(conns, &ConnectionDesc{Module: mod, OutIndex: outIndex, InIndex: inIndex})

		inIndexCnt++
	}

	return conns
}
