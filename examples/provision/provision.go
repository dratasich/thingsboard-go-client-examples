package main

import (
	"context"
	"os"
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

	// connect to TB
	provisionClient := tb.NewProvisioningClient(config.Thingsboard)
	provisionClient.Connect(ctx)

	// provision a device
	provisionClient.Provision("delete-me")

	// await response
	log.Debug().Msg("Waiting for device provisioning response...")
	res := <-provisionClient.ResponseQueue
	log.Info().Msgf("Received response: %+v", res)

	// try to connect with the new device
	config.Thingsboard.Username = res.CredentialsValue
	deviceClient := tb.NewClient(config.Thingsboard)
	deviceClient.Connect(ctx)

	// provision a device with access token
	token := "my-secret-token"
	provisionClient.ProvisionWithAccessToken("delete-me-2", token)

	// await response
	log.Debug().Msg("Waiting for device provisioning response...")
	res = <-provisionClient.ResponseQueue
	log.Info().Msgf("Received response: %+v", res)

	// try to connect with the new device
	config.Thingsboard.Username = token
	device2Client := tb.NewClient(config.Thingsboard)
	device2Client.Connect(ctx)

	// provision a device
	//
	// note: in a real world scenario, the certificate would be generated on the device itself and signed by a CA
	// create a certificate with:
	//   openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 -subj "/CN=delete-me" -keyout key.pem -out cert.pem
	// show the certificate:
	//   openssl x509 -in cert.pem -noout -text
	//
	// read from file for this example
	certPem, err := os.ReadFile("cert.pem")
	if err != nil {
		log.Fatal().Msgf("Failed to read certificate file: %s", err)
	}
	provisionClient.ProvisionWithCertificate("delete-me-2o", string(certPem))

	// await response
	log.Debug().Msg("Waiting for device provisioning response...")
	res = <-provisionClient.ResponseQueue
	log.Info().Msgf("Received response: %+v", res)

	// nice to have: connect with certificate to check if it worked
	// however, this requires a TLS connection which is not yet implemented in the SDK
	// also the server certs would need to be trusted

	// timeout the following shutdown process
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	// properly close connections on exit/sigint/sigterm
	provisionClient.Disconnect(shutdownCtx)
	deviceClient.Disconnect(shutdownCtx)
	device2Client.Disconnect(shutdownCtx)

	log.Info().Msg("Application stopped")
}
