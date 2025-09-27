package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"time"

	tb "github.com/dratasich/thingsboard-go-client-sdk"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"
)

// app configuration
type Config struct {
	Debug bool `env:"DEBUG,default=false"`

	// https://github.com/sethvargo/go-envconfig?tab=readme-ov-file#configuration
	Thingsboard tb.Config `env:", prefix=TB_MQTT_"`
}

func main() {
	// read config from env variables
	ctx := context.Background()
	var config Config
	if err := envconfig.Process(ctx, &config); err != nil {
		log.Fatal().Msgf("Failed to process envconfig: %s", err)
	}

	// setup logging (use zerolog global logger)
	// https://pkg.go.dev/github.com/rs/zerolog#section-readme
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if config.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Caller().
		Logger()
	log.Info().Msg("Application start...")

	// connect to TB (starts listening and processing RPCs)
	tbclient := tb.NewClient(config.Thingsboard)
	tbclient.Connect(ctx)

	// process rpc queue (endless)
	go func() {
		for {
			log.Debug().Msg("Waiting for RPC...")
			rpc := <-tbclient.RpcQueue
			log.Info().Msgf("Handling RPC request #%s %s: %s", rpc.RpcRequestId, rpc.Method, rpc.Params)

			var response_json []byte

			// check RPC method
			switch rpc.Method {
			// Ping
			case "ping":
				reply := "pong"
				response_json, _ = json.MarshalIndent(reply, "", " ")

			// all other RPC requests
			default:
				log.Error().Msgf("RPC Method unkown: %s", rpc.Method)
			}

			tbclient.ReplyRPC(rpc.RpcRequestId, response_json)
		}
	}()

	// wait for a signal to quit:
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	// timeout the following shutdown process
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	// properly close connections on exit/sigint/sigterm
	tbclient.Disconnect(shutdownCtx)

	log.Info().Msg("Application stopped")
}
