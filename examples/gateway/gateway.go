package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	tb "github.com/dratasich/thingsboard-go-client-sdk"
	"github.com/dratasich/thingsboard-go-client-sdk/events"
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
	tbclient := tb.NewGatewayClient(config.Thingsboard)
	tbclient.Connect(ctx)

	// loop and log gateway attribute updates
	go func() {
		for {
			log.Debug().Msg("Waiting for attribute response...")
			attrs := <-tbclient.AttributesQueue
			log.Info().Msgf("Received attribute update: %+v", attrs)
		}
	}()

	// loop and log gateway attribute updates
	go func() {
		for {
			log.Debug().Msg("Waiting for device attribute response...")
			attrs := <-tbclient.GatewayAttributesQueue
			log.Info().Msgf("Received attribute update: %+v", attrs)
		}
	}()

	// process rpc queue of the gateway itself (endless)
	go func() {
		for {
			log.Debug().Msg("Waiting for RPC...")
			rpc := <-tbclient.RpcQueue
			log.Info().Msgf("Handling RPC request #%d %s: %+v", rpc.RpcRequestId, rpc.Method, rpc.Params)

			// check RPC method
			switch rpc.Method {
			// ping
			case "gateway_ping":
				tbclient.ReplyGatewayPingRPC(rpc.RpcRequestId)
			// connected devices
			case "gateway_devices":
				tbclient.ReplyGatewayDevicesRPC(rpc.RpcRequestId)

			// all other RPC requests
			default:
				log.Error().Msgf("RPC Method unkown: %s", rpc.Method)
			}
		}
	}()

	// process rpc queue for connected devices (endless)
	go func() {
		for {
			log.Debug().Msg("Waiting for device RPC...")
			rpc := <-tbclient.GatewayRpcQueue
			log.Info().Msgf("Handling RPC request #%d %s for device %s: %+v", rpc.Data.RpcRequestId, rpc.Data.Method, rpc.Device, rpc.Data.Params)

			// check RPC method
			switch rpc.Data.Method {
			// when sent via the TB UI RPC widget, we get this method
			case "sendCommand":
				switch rpc.Data.Params.(map[string]any)["command"].(string) {
				case "ping":
					reply := events.GatewayResponseRPC{
						Device:    rpc.Device,
						RequestId: rpc.Data.RpcRequestId,
						Data:      "pong",
					}
					tbclient.ReplyDeviceRPC(reply)
				default:
					log.Error().Msgf("Device Command unkown: %+v", rpc.Data.Params)
				}

			// all other RPC requests
			default:
				log.Error().Msgf("RPC Method unkown: %s", rpc.Data.Method)
			}
		}
	}()

	tbclient.PublishTelemetry(events.Telemetry{
		Timestamp: time.Now().UnixMilli(),
		Values: map[string]any{
			"eventCounter": 1,
		},
	})

	// connect device (creates if not exists)
	tbclient.ConnectDevice("delete-me", "default")
	time.Sleep(1 * time.Second)

	// attributes request
	tbclient.RequestDeviceAttributes("delete-me", false, []string{"test"})
	attr := <-tbclient.GatewayAttributesResponseQueue
	log.Info().Msgf("Received attribute response #%d: %+v", attr.RequestId, attr)
	tbclient.RequestDeviceAttributes("delete-me", false, []string{"test", "timeout"})
	attr = <-tbclient.GatewayAttributesResponseQueue
	log.Info().Msgf("Received attribute response #%d: %+v", attr.RequestId, attr)

	// set client attributes
	tbclient.PublishDeviceAttributes("delete-me", map[string]any{
		"test": true,
	})

	telemetry := events.Telemetry{
		Timestamp: time.Now().UnixMilli(),
		Values: map[string]any{
			"test":         true,
			"eventCounter": 1,
		},
	}
	tbclient.SendTelemetry("delete-me", telemetry)

	time.Sleep(1 * time.Second)
	telemetry2 := events.Telemetry{
		Timestamp: time.Now().UnixMilli(),
		Values: map[string]any{
			"eventCounter": 2,
		},
	}
	time.Sleep(1 * time.Second)
	telemetry3 := events.Telemetry{
		Timestamp: time.Now().UnixMilli(),
		Values: map[string]any{
			"eventCounter": 3,
		},
	}
	batch := events.TelemetryBatch{
		"delete-me": []events.Telemetry{telemetry2, telemetry3},
	}
	tbclient.SendTelemetryBatch(batch)

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
