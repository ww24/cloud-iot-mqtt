package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/ww24/cloud-iot-mqtt/iot"
	"github.com/ww24/cloud-iot-mqtt/payload"
)

const (
	timeout           = 30 * time.Second
	protocolVersion   = 4 // MQTT 3.1.1
	clientIDFormat    = "projects/%s/locations/%s/registries/%s/devices/%s"
	jwtExpireDuration = time.Hour // Max 24 hours.
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
	endpoint    = os.Getenv("ENDPOINT")

	c iot.CloudIotClient
)

func main() {
	clientID := fmt.Sprintf(clientIDFormat, projectID, cloudRegion, registoryID, deviceID)
	log.Printf("Broker: %s, ClientID: %s\n", broker, clientID)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(timeout)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(false)
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
		ExpiresAt: now.Add(jwtExpireDuration).Unix(),
		Audience:  projectID,
	})
	password, err := t.SignedString(cert.PrivateKey)
	if err != nil {
		log.Fatalf("Error: %+v\n", err)
	}
	opts.SetPassword(password)

	opts.SetConnectionLostHandler(func(cli mqtt.Client, err error) {
		log.Printf("Connection lost reason: %+v\n", err)

		// reconnect
		now := time.Now()
		t := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(jwtExpireDuration).Unix(),
			Audience:  projectID,
		})
		password, err := t.SignedString(cert.PrivateKey)
		if err != nil {
			log.Fatalf("Error: %+v\n", err)
		}
		opts.SetPassword(password)
		connect(opts)
	})

	opts.SetOnConnectHandler(func(cli mqtt.Client) {
		go func() {
			ticker := time.NewTicker(time.Minute)
			for {
				select {
				case t := <-ticker.C:
					log.Printf("connected: %v, ts: %s \n", cli.IsConnected(), t)
				}
			}
		}()

		{
			token := cli.Subscribe(fmt.Sprintf(iot.TopicFormat, deviceID, "config"), qosAtLeastOnce, func(client mqtt.Client, msg mqtt.Message) {
				log.Printf("config:: topic: %s, payload: %s\n", msg.Topic(), string(msg.Payload()))
			})
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
		}
		{
			token := cli.Subscribe(fmt.Sprintf(iot.TopicFormat, deviceID, "state"), qosAtLeastOnce, func(client mqtt.Client, msg mqtt.Message) {
				log.Printf("state:: topic: %s, payload: %s\n", msg.Topic(), string(msg.Payload()))
			})
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
		}
		{
			// https://cloud.google.com/iot/docs/how-tos/commands?hl=ja
			token := cli.Subscribe(fmt.Sprintf(iot.TopicFormat, deviceID, "commands")+"/#", qosAtLeastOnce, func(client mqtt.Client, msg mqtt.Message) {
				log.Printf("commands:: topic: %s, payload: %s\n", msg.Topic(), string(msg.Payload()))
				if strings.HasSuffix(msg.Topic(), "commands/signal") {
					payload := &payload.Payload{}
					if err := json.Unmarshal(msg.Payload(), payload); err != nil {
						log.Println("Err:", err)
						return
					}
					log.Println("call", endpoint)
					resp, err := http.Post(endpoint, "application/json", bytes.NewReader(msg.Payload()))
					if err != nil {
						log.Println("Err:", err)
						return
					}
					defer resp.Body.Close()
					res, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Println("Err:", err)
						return
					}
					log.Println("Response:", string(res))
				}
			})
			if token.Wait() && token.Error() != nil {
				log.Fatal(token.Error())
			}
		}
	})

	connect(opts)
	cli := c.Client()
	defer c.UpdateState(deviceID, "stopped")
	defer cli.Disconnect(250)

	// c.PublishEvent(deviceID, "button")

	// ticker := time.NewTicker(time.Second * 5)
	// defer ticker.Stop()
	// c.HeartBeat(deviceID, ticker)

	signalHandler()
}

func connect(opts *mqtt.ClientOptions) {
	c = iot.NewCloudIotClient(opts)
	if err := c.Connect(); err != nil {
		log.Fatalf("Error: %+v\n", err)
	}

	log.Println("CONNECTED!")
	c.UpdateState(deviceID, "started")
}

func signalHandler() {
	ch := make(chan os.Signal, 0)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-ch:
		log.Printf("signal received: %s\n", sig)
	}
	os.Exit(0)
}
