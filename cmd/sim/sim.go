package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/andrew-r-thomas/mqtt/users"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const numUsers = 5
const locFreq = time.Second
const brokerAddr = ":1883"

func main() {
	logger := zerolog.New(os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = logger.WithContext(ctx)

	var wg sync.WaitGroup
	wg.Add(numUsers)
	for range numUsers {
		time.Sleep(time.Second)

		user := users.TypicalUser{
			Id:         uuid.NewString(),
			LocFreq:    locFreq,
			BrokerAddr: brokerAddr,
		}
		go user.Run(ctx, &wg)
	}

	time.Sleep(time.Second * 10)
	cancel()
	wg.Wait()
}
