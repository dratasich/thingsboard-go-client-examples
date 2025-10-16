# thingsboard-go-client-examples

Examples using the [thingsboard-go-client-sdk](https://github.com/dratasich/thingsboard-go-client-sdk).

## Run

```bash
cd examples/rpc
go mod tidy
DEBUG=true TB_MQTT_SERVER_URL=mqtts://<host>:8883 TB_MQTT_USERNAME=<access token> go run .
```

### gateway

When running the `gateway`-example be sure you connect to a device with gateway-mode enabled.

### provision

Create a device profile and select `Allow to create devices` in the "Device Provisioning" tab.

```bash
DEBUG=true TB_MQTT_SERVER_URL=mqtts://<host>:8883 TB_MQTT_PROVISIONING_KEY=<key> TB_MQTT_PROVISIONING_SECRET=<secret> go run .
```
