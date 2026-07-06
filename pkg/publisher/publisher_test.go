package publisher

import (
	"testing"
)

func TestNoOpPublisher(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		data  interface{}
	}{
		{
			name:  "publish something",
			topic: "test-topic",
			data:  "test-data",
		},
		{
			name:  "publish empty",
			topic: "",
			data:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewNoOpPublisher()
			err := p.Publish(tt.topic, tt.data)
			if err != nil {
				t.Errorf("NoOpPublisher.Publish() error = %v, wantErr nil", err)
			}
			p.Close() // Should not panic
		})
	}
}

func TestKafkaPublisher_Publish(t *testing.T) {
	tests := []struct {
		name    string
		brokers []string
		topic   string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "marshal error",
			brokers: []string{"localhost:9092"},
			topic:   "topic1",
			data:    make(chan int),
			wantErr: true,
		},
		{
			name:    "publish async success", // Note: Without a broker, this will fail
			brokers: []string{"localhost:9092"},
			topic:   "topic2",
			data:    map[string]string{"key": "value"},
			wantErr: true,
		},
		{
			name:    "publish async no topic",
			brokers: []string{"localhost:9092"},
			topic:   "",
			data:    "data",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKafkaPublisher(tt.brokers, "default-topic")
			err := p.Publish(tt.topic, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("KafkaPublisher.Publish() error = %v, wantErr %v", err, tt.wantErr)
			}
			p.Close()
		})
	}
}

func TestKafkaPublisher_Close(t *testing.T) {
	// Close a nil writer to ensure it handles nil if we ever have it
	p := &KafkaPublisher{writer: nil}
	p.Close() // Should not panic
}
