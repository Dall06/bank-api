package publisher

// Publisher define la interfaz general para publicar eventos a un broker de mensajería.
type Publisher interface {
	Publish(topic string, data interface{}) error
	Close()
}

// NoOpPublisher es un publicador de eventos vacío (no hace nada).
// Es muy útil para pruebas unitarias o entornos locales sin un broker activo.
type NoOpPublisher struct{}

// NewNoOpPublisher crea un nuevo NoOpPublisher.
func NewNoOpPublisher() *NoOpPublisher {
	return &NoOpPublisher{}
}

func (p *NoOpPublisher) Publish(topic string, data interface{}) error {
	return nil
}

func (p *NoOpPublisher) Close() {}
