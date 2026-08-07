package psalm

type Socket struct {
	Channels    Channels
	Connections []*Connection
}

func (s *Socket) AddConnection(conn *Connection) {
	s.Connections = append(s.Connections, conn)
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
