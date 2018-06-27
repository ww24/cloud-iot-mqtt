package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	timeout         = 30 * time.Second
	protocolVersion = 4 // MQTT 3.1.1
	clientIDFormat  = "projects/%s/locations/%s/registries/%s/devices/%s"
	topicFormat     = "/devices/%s/%s"
)

const (
	qosAtMostOnce byte = iota
	qosAtLeastOnce
	qosExactlyOnce
)

var (
	broker      = os.Getenv("BROKER")
	projectID   = os.Getenv("PROJECT_ID")
	cloudRegion = os.Getenv("CLOUD_REGION")
	registoryID = os.Getenv("REGISTORY_ID")
	deviceID    = os.Getenv("DEVICE_ID")
)

func main() {
	clientID := fmt.Sprintf(clientIDFormat, projectID, cloudRegion, registoryID, deviceID)
	log.Printf("Broker: %s, ClientID: %s\n", broker, clientID)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(timeout)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetProtocolVersion(protocolVersion)
	opts.SetStore(mqtt.NewMemoryStore())

	// Set Root CA certificate (optional)
	data, err := ioutil.ReadFile("roots.pem")
	if err != nil {
		log.Printf("Warn: %+v\n", err)
	} else {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			log.Fatalf("Error: failed to append root ca")
		}
		opts.SetTLSConfig(&tls.Config{
			RootCAs: pool,
		})
	}

	opts.SetUsername("unused")

	cert, err := tls.LoadX509KeyPair("rsa_cert.pem", "rsa_private.pem")
	if err != nil {
		log.Fatalf("Error: %+v\n", err)
	}
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
		Audience:  projectID,
	})
	password, err := t.SignedString(cert.PrivateKey)
	if err != nil {
		log.Fatalf("Error: %+v\n", err)
	}
	opts.SetPassword(password)

	opts.SetOnConnectHandler(func(cli mqtt.Client) {
		{
			token := cli.Subscribe(fmt.Sprintf(topicFormat, deviceID, "config"), qosAtLeastOnce, func(client mqtt.Client, msg mqtt.Message) {
				log.Printf("topic: %s, payload: %s\n", msg.Topic(), string(msg.Payload()))
			})
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
		}
		{
			token := cli.Subscribe(fmt.Sprintf(topicFormat, deviceID, "state"), qosAtLeastOnce, func(client mqtt.Client, msg mqtt.Message) {
				log.Printf("topic: %s, payload: %s\n", msg.Topic(), string(msg.Payload()))
			})
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
		}
	})

	cli := mqtt.NewClient(opts)
	defer cli.Disconnect(250)

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error: %+v\n", token.Error())
	}

	log.Println("CONNECTED!")
	updateState(cli, "started")
	defer func() {
		updateState(cli, "stopped")
	}()

	{
		topic := fmt.Sprintf(topicFormat, deviceID, "events/button")
		token := cli.Publish(topic, qosAtLeastOnce, false, "on")
		if token.Wait() && token.Error() != nil {
			log.Fatalf("Error: %+v\n", token.Error())
		}
	}

	signalHandler()
}

func signalHandler() {
	ch := make(chan os.Signal, 0)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-ch:
		log.Printf("signal received: %s\n", sig)
	}
}

func updateState(cli mqtt.Client, state string) {
	topic := fmt.Sprintf(topicFormat, deviceID, "state")
	token := cli.Publish(topic, qosAtLeastOnce, false, state)
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Error: %+v\n", token.Error())
	}
}
