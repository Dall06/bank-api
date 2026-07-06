package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// KafkaPublisher es el adaptador concreto de Kafka que implementa la interfaz Publisher.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher inicializa un publicador asíncrono y no bloqueante para Kafka.
func NewKafkaPublisher(brokers []string, defaultTopic string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        defaultTopic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll, // acks=all para máxima seguridad transaccional
			Async:        true,             // Asíncrono para no bloquear los hilos HTTP
		},
	}
}

// Publish envía un mensaje serializado en JSON al tópico indicado.
// Si topic es vacío, utiliza el tópico por defecto configurado.
func (p *KafkaPublisher) Publish(topic string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("kafka marshal: %w", err)
	}

	msg := kafka.Message{
		Value: payload,
	}

	if topic != "" {
		msg.Topic = topic
	}

	// WriteMessages asíncrono
	if err := p.writer.WriteMessages(context.Background(), msg); err != nil {
		return fmt.Errorf("kafka publish to %s: %w", topic, err)
	}

	return nil
}

// Close libera las conexiones del productor con el clúster.
func (p *KafkaPublisher) Close() {
	if p.writer != nil {
		_ = p.writer.Close()
	}
}
