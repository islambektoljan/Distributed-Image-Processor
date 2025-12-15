package rabbitmq

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const QueueName = "image_processing_queue"

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewClient creates a new RabbitMQ client and declares the queue
func NewClient(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queue with durable flag
	_, err = ch.QueueDeclare(
		QueueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	log.Printf("RabbitMQ client initialized successfully with queue: %s", QueueName)

	return &Client{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish sends a message to the queue
func (c *Client) Publish(body []byte) error {
	err := c.channel.Publish(
		"",        // exchange
		QueueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message to queue: %s", QueueName)
	return nil
}

// Consume starts consuming messages from the queue
func (c *Client) Consume() (<-chan amqp.Delivery, error) {
	// Set QoS to limit number of unacknowledged messages
	err := c.channel.Qos(
		5,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := c.channel.Consume(
		QueueName, // queue
		"",        // consumer tag
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Started consuming messages from queue: %s", QueueName)
	return msgs, nil
}

// Close closes the channel and connection
func (c *Client) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}
	return nil
}
