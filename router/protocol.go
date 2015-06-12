package router

const (
	Protocol        = "weave"
	ProtocolVersion = 17
)

type ProtocolTag byte

const (
	ProtocolHeartbeat ProtocolTag = iota
	ProtocolInterHostControlMsg
	ProtocolGossip
	ProtocolGossipUnicast
	ProtocolGossipBroadcast
)

type ProtocolMsg struct {
	tag ProtocolTag
	msg []byte
}

type ProtocolSender interface {
	SendProtocolMsg(m ProtocolMsg)
}
