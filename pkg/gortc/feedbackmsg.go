package gortc

import "github.com/let-light/neon/pkg/forwarder"

type RetransmitPacketMsg struct {
	Dest    forwarder.IFrameDestination
	Packets []packetMeta
}
