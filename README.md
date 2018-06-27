# Cloud IoT MQTT example for golang

## References
- https://cloud.google.com/iot/docs/how-tos/mqtt-bridge
- https://cloud.google.com/iot/docs/how-tos/credentials/keys

## Usage
- `copy env.mk.sample env.mk` and fill variables.
- `openssl req -x509 -nodes -newkey rsa:2048 -keyout rsa_private.pem -days 1000000 -out rsa_cert.pem -subj "/CN=unused"`
- set up IoT Core device on [Google Cloud Console](https://console.cloud.google.com/iot/)
- `make run`
