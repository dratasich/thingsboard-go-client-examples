package main

import (
	"context"
	"os"
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

	// connect to TB
	tbclient := tb.NewClient(config.Thingsboard)
	tbclient.Connect(ctx)

	// publish telemetry
	telemetry := tbevents.Telemetry{
		Timestamp: time.Now().UnixMilli(),
		Values: map[string]any{
			"temperature": 25.3,
			"humidity":    60,
		},
	}
	tbclient.PublishTelemetry(telemetry)

	// timeout the following shutdown process
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	// properly close connections on exit/sigint/sigterm
	tbclient.Disconnect(shutdownCtx)

	log.Info().Msg("Application stopped")
}
