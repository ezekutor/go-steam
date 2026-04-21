package mirror

import (
	"context"
	"errors"
	"sync"
	"time"

	steam "github.com/Philipp15b/go-steam/v3"
	"github.com/Philipp15b/go-steam/v3/mirror/player"
	"github.com/Philipp15b/go-steam/v3/mirror/server"
)

// startPlayersForServer launches a steam client for each credential pair
// attached to the provided server. Each account is started sequentially and
// the function waits for a LoggedOnEvent before proceeding to the next
// account. The resulting player instances are returned to the caller and also
// tracked globally through playersByServer.
func startPlayersForServer(ctx context.Context, srv *server.Server) ([]*player.Player, error) {
	if srv == nil {
		return nil, errors.New("mirror: server must not be nil")
	}

	creds := srv.Accounts()
	players := make([]*player.Player, 0, len(creds))

	for _, cred := range creds {
		loggedOn := make(chan *steam.LoggedOnEvent, 1)
		pl, err := startPlayer(ctx, srv, cred, func(evt *steam.LoggedOnEvent) {
			select {
			case loggedOn <- evt:
			default:
			}
		})
		if err != nil {
			return nil, err
		}

		select {
		case <-ctx.Done():
			pl.Logoff()
			return nil, ctx.Err()
		case evt, ok := <-loggedOn:
			if !ok || evt == nil {
				pl.Logoff()
				return nil, errors.New("mirror: failed to confirm logon")
			}
		}

		time.Sleep(10 * time.Millisecond)
		players = append(players, pl)
	}

	return players, nil
}

// startPlayer creates a player backed by a new Steam client and begins the
// connection process. When the account successfully logs on the provided
// callback is invoked with the LoggedOnEvent instance.
func startPlayer(ctx context.Context, srv *server.Server, cred server.Credentials, onLoggedOn func(*steam.LoggedOnEvent)) (*player.Player, error) {
	_ = srv
	client := steam.NewClient()
	pl := player.New(client)

	events := client.Events()
	var once sync.Once
	notify := func(evt *steam.LoggedOnEvent) {
		once.Do(func() {
			if onLoggedOn != nil {
				onLoggedOn(evt)
			}
		})
	}

	go func() {
		defer notify(nil)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}

				switch e := event.(type) {
				case *steam.ConnectedEvent:
					client.Auth.LogOn(&steam.LogOnDetails{
						Username: cred.Login,
						Password: cred.Password,
					})
				case *steam.LoggedOnEvent:
					notify(e)
				case *steam.LogOnFailedEvent:
					notify(nil)
					return
				case *steam.FatalErrorEvent, *steam.DisconnectedEvent:
					return
				}
			}
		}
	}()

	if _, err := client.Connect(); err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			pl.Logoff()
		}
	}()

	return pl, nil
}
