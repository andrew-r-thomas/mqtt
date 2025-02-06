package users

import (
	"context"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type TypicalUser struct {
	Id         string
	LocFreq    time.Duration
	BrokerAddr string
}

func (t *TypicalUser) Run(ctx context.Context, wg *sync.WaitGroup) {
	done := ctx.Done()
	log := zerolog.Ctx(ctx)

	// connect
	conn, err := net.Dial("tcp", t.BrokerAddr)
	if err != nil {
		log.Fatal().Err(err).Str("id", t.Id).Msg("error dialing broker")
	}

	client := paho.NewClient(
		paho.ClientConfig{
			OnPublishReceived: []func(
				p paho.PublishReceived,
			) (bool, error){
				func(p paho.PublishReceived) (bool, error) {
					time.Sleep(time.Millisecond)
					ts := time.Now()
					pid := string(p.Packet.Payload[:36])
					log.Info().
						Str("id", t.Id).
						Str("pid", pid).
						Time("time", ts).
						Msg("publish recieved")

					return true, nil
				},
			},
			Conn:     conn,
			ClientID: t.Id,
		},
	)

	_, err = client.Connect(
		ctx,
		&paho.Connect{
			ClientID:     t.Id,
			KeepAlive:    30,
			UsernameFlag: false,
			PasswordFlag: false,
			CleanStart:   true,
		},
	)
	if err != nil {
		log.Fatal().Err(err).Str("id", t.Id).Msg("error connecting")
	}

	// subscribe
	_, err = client.Subscribe(
		ctx,
		&paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{
					Topic:   "test",
					QoS:     0,
					NoLocal: true,
				},
			},
		},
	)
	if err != nil {
		log.Fatal().Err(err).Str("id", t.Id).Msg("error subscribing")
	}

	ticker := time.NewTicker(t.LocFreq)
	for {
		select {
		case <-done:
			// normal disconnect
			err = client.Disconnect(
				&paho.Disconnect{ReasonCode: 0},
			)
			if err != nil {
				log.Fatal().
					Err(err).
					Str("id", t.Id).
					Msg("error disconnecting")
			}
			wg.Done()
			return
		case <-ticker.C:
			// periodic publish
			pid := uuid.NewString()
			lat := rand.Float64()
			long := rand.Float64()

			payload := make([]byte, 52)
			copy(payload, pid)
			binary.BigEndian.PutUint64(
				payload[36:44],
				math.Float64bits(lat),
			)
			binary.BigEndian.PutUint64(
				payload[44:52],
				math.Float64bits(long),
			)
			pub := &paho.Publish{
				Topic:   "test",
				QoS:     0,
				Payload: payload,
			}

			log.Info().
				Str("id", t.Id).
				Str("pid", pid).
				Timestamp().
				Msg("publishing")
			_, err := client.Publish(ctx, pub)
			if err != nil {
				log.Fatal().
					Err(err).
					Str("id", t.Id).
					Str("pid", pid).
					Msg("error publishing")
			}
		}
	}
}
