package mqttserver

import (
	"fmt"
	"go-iot/codec"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func ClientStart(network codec.Network) error {
	spec := MQTTClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(spec.ClientId)
	opts.SetUsername(spec.Username)
	opts.SetPassword(spec.Password)
	opts.SetCleanSession(spec.CleanSession)

	choke := make(chan MQTT.Message)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- msg
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	session := ClientSession{
		client:    client,
		ClientID:  spec.ClientId,
		Username:  spec.Username,
		CleanFlag: spec.CleanSession,
		Topics:    spec.Topics,
	}

	// create codec
	c := codec.NewCodec(network)

	go func() {
		c.OnConnect(&mqttClientContext{
			BaseContext: codec.BaseContext{
				DeviceId:  session.GetDeviceId(),
				ProductId: network.ProductId,
				Session:   &session,
			},
		})
		for {
			msg := <-choke
			c.OnMessage(&mqttClientContext{
				BaseContext: codec.BaseContext{
					DeviceId:  session.GetDeviceId(),
					ProductId: network.ProductId,
					Session:   &session,
				},
				Data: msg,
			})
		}
	}()
	return nil
}
