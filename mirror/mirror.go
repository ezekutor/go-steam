package mirror

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Philipp15b/go-steam/v3/mirror/player"
	"github.com/Philipp15b/go-steam/v3/mirror/server"
)

var (
	playersMu       sync.Mutex
	playersByServer = make(map[*server.Server][]*player.Player)
)

// runMirror launches the mirror workers for the provided server and keeps
// them refreshed on a fixed cadence. Players are restarted every six minutes
// to ensure that authentication tokens remain up-to-date.
func runMirror(ctx context.Context, srv *server.Server) error {
	if srv == nil {
		return errors.New("mirror: server must not be nil")
	}

	players, err := startPlayersForServer(ctx, srv)
	if err != nil {
		return err
	}

	setPlayersForServer(srv, players)

	ticker := time.NewTicker(6 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logoffPlayers(players)
			clearPlayersForServer(srv)
			return ctx.Err()
		case <-ticker.C:
			logoffPlayers(players)
			clearPlayersForServer(srv)

			players, err = startPlayersForServer(ctx, srv)
			if err != nil {
				return err
			}

			setPlayersForServer(srv, players)
		}
	}
}

func setPlayersForServer(srv *server.Server, players []*player.Player) {
	playersMu.Lock()
	defer playersMu.Unlock()
	playersByServer[srv] = players
}

func clearPlayersForServer(srv *server.Server) {
	playersMu.Lock()
	defer playersMu.Unlock()
	delete(playersByServer, srv)
}

func logoffPlayers(players []*player.Player) {
	for _, p := range players {
		if p != nil {
			p.Logoff()
		}
	}
}
