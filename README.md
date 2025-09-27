# thingsboard-go-client-examples

Examples using the [thingsboard-go-client-sdk](https://github.com/dratasich/thingsboard-go-client-sdk).

## Run

```bash
cd examples/rpc
go mod tidy
DEBUG=true TB_MQTT_SERVER_URL=mqtts://<host>:8883 TB_MQTT_USERNAME=<access token> go run .
```
