package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	tb "github.com/dratasich/thingsboard-go-client-sdk"
	tbevents "github.com/dratasich/thingsboard-go-client-sdk/events"
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

	// set client attributes
	clientAttr := tbevents.Attributes{
		"serial": "123",
	}
	tbclient.PublishAttributes(clientAttr)

	// request attributes
	request := tbevents.RequestAttributes{}
	request.ClientKeys = "serial"
	request.SharedKeys = "name,timeout"
	tbclient.RequestAttributes(request)

	// await response
	log.Debug().Msg("Waiting for attribute response...")
	attr := <-tbclient.AttributesResponseQueue
	// print result
	log.Info().Msgf("Received attribute response #%d: %+v", attr.RequestId, attr)
	print("client attributes:\n")
	for key, value := range *attr.ClientAttr {
		print(fmt.Sprintf("  %s = %+v\n", key, value))
	}
	print("shared attributes:\n")
	for key, value := range *attr.SharedAttr {
		print(fmt.Sprintf("  %s = %+v\n", key, value))
	}

	// loop and log attribute updates
	go func() {
		for {
			log.Debug().Msg("Waiting for attribute response...")
			attrs := <-tbclient.AttributesQueue
			log.Info().Msgf("Received attribute update...")
			print("attributes:\n")
			for key, value := range *attrs {
				print(fmt.Sprintf("  %s = %s\n", key, value))
			}
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
