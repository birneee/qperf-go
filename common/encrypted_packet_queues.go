package common

import (
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"sync"
)

type EncryptedPacketQueues struct {
	mutex           sync.Mutex
	pendingRestores map[string] /* connection id */ []quic.UnhandledPacket
}

// TODO add maximum
// TODO add timeout
func NewEncryptedPacketQueues() *EncryptedPacketQueues {
	return &EncryptedPacketQueues{
		pendingRestores: map[string][]quic.UnhandledPacket{},
	}
}

// true if new entry for connection id was created
func (p *EncryptedPacketQueues) Enqueue(connID logging.ConnectionID, packet quic.UnhandledPacket) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	queue, ok := p.pendingRestores[connID.String()]
	if !ok {
		queue = []quic.UnhandledPacket{}
		p.pendingRestores[connID.String()] = queue
	}
	p.pendingRestores[connID.String()] = append(queue, packet)
	return !ok
}

func (p *EncryptedPacketQueues) Dequeue(connID logging.ConnectionID, conn quic.Connection) []quic.UnhandledPacket {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	queue := p.pendingRestores[connID.String()]
	for _, packet := range queue {
		conn.HandlePacket(packet)
	}
	delete(p.pendingRestores, connID.String())
	return queue
}
