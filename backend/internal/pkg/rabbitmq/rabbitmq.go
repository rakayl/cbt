package rabbitmq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func New(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Publisher{Conn: conn, Channel: ch}, nil
}
func (p *Publisher) Publish(ctx context.Context, queue string, body []byte) error {
	_, err := p.Channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	return p.Channel.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: body})
}
func (p *Publisher) Close() error {
	if p.Channel != nil {
		_ = p.Channel.Close()
	}
	if p.Conn != nil {
		return p.Conn.Close()
	}
	return nil
}
