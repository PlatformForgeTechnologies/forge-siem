package response

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"forge-siem/internal/types"
)

const (
	alertsExchange = "siem.alerts"
)

func setupBroker(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(alertsExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	type queueBinding struct {
		name       string
		routingKey string
	}
	bindings := []queueBinding{
		{name: "active-response", routingKey: "severity.critical.#"},
		{name: "notifications", routingKey: "severity.#"},
		{name: "webhooks", routingKey: "#"},
	}

	for _, binding := range bindings {
		if _, err := channel.QueueDeclare(binding.name, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare queue %s: %w", binding.name, err)
		}
		if err := channel.QueueBind(binding.name, binding.routingKey, alertsExchange, false, nil); err != nil {
			return fmt.Errorf("bind queue %s: %w", binding.name, err)
		}
	}
	return nil
}

func publishAlert(channel *amqp.Channel, alert types.Alert) error {
	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}
	routingKey := "severity." + alert.Severity + "." + alert.AgentID
	return channel.Publish(alertsExchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
