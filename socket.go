package psalm

type Socket struct {
	Channels    Channels
	Connections []*Connection
}

func (s *Socket) AddConnection(conn *Connection) {
	s.Connections = append(s.Connections, conn)
}

// RemoveConnection takes the connections with a module and index from the socket, and tells
// whether there were any. The list is replaced, not changed in place, so a loop over it that is
// running goes on with the list it had
func (s *Socket) RemoveConnection(module Modular, index int) bool {
	kept := make([]*Connection, 0, len(s.Connections))
	for _, c := range s.Connections {
		if c.Module != module || c.Index != index {
			kept = append(kept, c)
		}
	}
	removed := len(kept) != len(s.Connections)
	if removed {
		s.Connections = kept
	}
	return removed
}

func (s *Socket) ProcessIncoming() {
	s.Channels.Clear()
	for _, conn := range s.Connections {
		s.Channels.Accum(conn.OutputChannels())
	}
}

func (s *Socket) IsConnected() bool {
	return len(s.Connections) > 0
}

func (s *Socket) Copy(other *Socket) {
	s.Channels.Copy(other.Channels)
}

func (s *Socket) Clear() {
	s.Channels.Clear()
}

func NumSockets(num int, numChannels int) []int {
	numSockets := make([]int, num)
	for i := 0; i < num; i++ {
		numSockets[i] = numChannels
	}
	return numSockets
}
