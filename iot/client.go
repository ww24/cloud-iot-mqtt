package iot

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	TopicFormat = "/devices/%s/%s"
)

const (
	QosAtMostOnce byte = iota
	QosAtLeastOnce
	QosExactlyOnce
)

// CloudIotClient represents mqtt.Client wrapper.
type CloudIotClient interface {
	Connect() error
	Client() mqtt.Client
	HeartBeat(deviceID string, ticker *time.Ticker)
	UpdateState(deviceID, state string) error
	PublishEvent(deviceID, eventName string) error
}

type cloudIotClient struct {
	client mqtt.Client
}

// NewCloudIotClient returns mqtt client.
func NewCloudIotClient(opts *mqtt.ClientOptions) CloudIotClient {
	if opts == nil {
		opts = mqtt.NewClientOptions()
	}

	return &cloudIotClient{
		client: mqtt.NewClient(opts),
	}
}

func (c *cloudIotClient) Connect() error {
	token := c.client.Connect()
	token.Wait()
	return token.Error()
}

func (c *cloudIotClient) Client() mqtt.Client {
	return c.client
}

func (c *cloudIotClient) HeartBeat(deviceID string, ticker *time.Ticker) {
	for t := range ticker.C {
		log.Println("timestamp", t)
		c.UpdateState(deviceID, "heartBeat")
	}
}

func (c *cloudIotClient) UpdateState(deviceID, state string) error {
	topic := fmt.Sprintf(TopicFormat, deviceID, "state")
	token := c.client.Publish(topic, QosAtLeastOnce, false, state)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *cloudIotClient) PublishEvent(deviceID, eventName string) error {
	topic := fmt.Sprintf(TopicFormat, deviceID, "events/"+eventName)
	token := c.client.Publish(topic, QosAtLeastOnce, false, "on")
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
