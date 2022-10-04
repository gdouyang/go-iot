package mqttserver

import (
	"fmt"
	"go-iot/codec"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func ClientStart(network codec.Network) {
	spec := MQTTClientSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port
	opts := MQTT.NewClientOptions()
	opts.AddBroker("tcp://" + spec.Host + ":" + fmt.Sprint(spec.Port))
	opts.SetClientID(spec.ClientId)
	opts.SetUsername(spec.Username)
	opts.SetPassword(spec.Password)
	opts.SetCleanSession(spec.CleanSession)
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

}
