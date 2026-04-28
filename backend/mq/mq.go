package mq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/cinema-booking/backend/models"
	"github.com/cinema-booking/backend/repository"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueBookingEvents = "booking_events"
)

type Producer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewProducer(url string) (*Producer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	_, err = ch.QueueDeclare(QueueBookingEvents, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return &Producer{conn: conn, ch: ch}, nil
}

func (p *Producer) Publish(eventType string, payload map[string]interface{}) error {
	event := models.MQEvent{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(context.Background(), "", QueueBookingEvents, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (p *Producer) Close() {
	p.ch.Close()
	p.conn.Close()
}

// ---

type Consumer struct {
	conn        *amqp.Connection
	ch          *amqp.Channel
	auditRepo   *repository.AuditLogRepository
}

func NewConsumer(url string, auditRepo *repository.AuditLogRepository) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	_, err = ch.QueueDeclare(QueueBookingEvents, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return &Consumer{conn: conn, ch: ch, auditRepo: auditRepo}, nil
}

func (c *Consumer) Start() {
	msgs, err := c.ch.Consume(QueueBookingEvents, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("[MQ] Failed to start consumer: %v", err)
		return
	}

	log.Println("[MQ] Consumer started, waiting for events...")

	go func() {
		for msg := range msgs {
			var event models.MQEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("[MQ] Failed to parse event: %v", err)
				msg.Nack(false, false)
				continue
			}

			c.handle(event)
			msg.Ack(false)
		}
	}()
}

func (c *Consumer) handle(event models.MQEvent) {
	ctx := context.Background()
	log.Printf("[MQ] Received event: %s | payload: %+v", event.Type, event.Payload)

	switch event.Type {
	case "booking.completed":
		// Write audit log
		if err := c.auditRepo.Create(ctx, event.Type, event.Payload); err != nil {
			log.Printf("[MQ] Failed to write audit log: %v", err)
		}
		// Mock notification
		log.Printf("[NOTIFY] Booking confirmed for user %v, seat %v", event.Payload["user_email"], event.Payload["seat_id"])

	case "booking.timeout":
		if err := c.auditRepo.Create(ctx, event.Type, event.Payload); err != nil {
			log.Printf("[MQ] Failed to write audit log: %v", err)
		}
		log.Printf("[NOTIFY] Booking timed out for seat %v", event.Payload["seat_id"])

	case "seat.released":
		if err := c.auditRepo.Create(ctx, event.Type, event.Payload); err != nil {
			log.Printf("[MQ] Failed to write audit log: %v", err)
		}

	default:
		log.Printf("[MQ] Unknown event type: %s", event.Type)
	}
}

func (c *Consumer) Close() {
	c.ch.Close()
	c.conn.Close()
}
