package player

import (
	"sync"
	"time"

	steam "github.com/Philipp15b/go-steam/v3"
	"github.com/Philipp15b/go-steam/v3/protocol"
	"github.com/Philipp15b/go-steam/v3/protocol/protobuf"
	"github.com/Philipp15b/go-steam/v3/protocol/steamlang"
)

// Player represents a lightweight wrapper around a Steam client that
// provides lifecycle helpers for mirroring accounts.
type Player struct {
	client *steam.Client
	once   sync.Once
}

// New constructs a new Player instance bound to the provided Steam client.
func New(client *steam.Client) *Player {
	return &Player{client: client}
}

// Client returns the underlying Steam client instance.
func (p *Player) Client() *steam.Client {
	return p.client
}

// Logoff gracefully logs the player out of Steam. It emits a
// CMsgClientLogOff message and then closes the connection to the Steam
// network.
func (p *Player) Logoff() {
	if p == nil {
		return
	}

	p.once.Do(func() {
		client := p.client
		if client == nil {
			return
		}

		client.Write(protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientLogOff, new(protobuf.CMsgClientLogOff)))
		// Give Steam a brief moment to process the logoff before tearing
		// down the TCP connection. This mirrors the behaviour of
		// Client.Disconnect which performs a similar grace period.
		time.Sleep(250 * time.Millisecond)
		client.Disconnect()
	})
}
