package rabbitmq

import (
	"sync"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type RabbitMq struct {
	URI        string
	Queue      string
	Exchange   string
	Channel    *amqp.Channel
	Connection *amqp.Connection
}

var RabbitMQInstance *RabbitMq

var once sync.Once

func NewRabbitMq(uri, queue, exchange string) *RabbitMq {
	once.Do(func() {
		rabbitMq := &RabbitMq{
			URI:        uri,
			Queue:      queue,
			Exchange:   exchange,
			Channel:    nil,
			Connection: nil,
		}
		RabbitMQInstance = rabbitMq
	})

	return RabbitMQInstance
}

func (r *RabbitMq) SetupRabbitMq() {
	conn, ch, err := r.Connect()

	if err != nil {
		zap.S().Errorf("failed to connect to rabbitmq: %s", err)
	}

	r.Channel = ch
	r.Connection = conn
	// defer conn.Close()
	// defer ch.Close()

	err = r.DeclareExchange(ch)
	if err != nil {
		zap.S().Errorf("failed to declare exchange: %s", err)
	}

	_, err = r.DeclareQueue(ch)
	if err != nil {
		zap.S().Errorf("failed to declare queue: %s", err)
	}

	err = r.BindQueue(ch)
	if err != nil {
		zap.S().Errorf("failed to bind queue: %s", err)
	}

	zap.S().Infof("connected to rabbitmq at %s with queue %s and exchange %s", r.URI, r.Queue, r.Exchange)

}

func (r *RabbitMq) Connect() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(r.URI)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

func (r *RabbitMq) Close(conn *amqp.Connection, ch *amqp.Channel) {
	r.Connection.Close()
	r.Channel.Close()
}

func (r *RabbitMq) DeclareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		r.Exchange, // name
		"direct",   // type
		true,       // durable
		false,      // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
}

func (r *RabbitMq) DeclareQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		r.Queue, // queue name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
}

func (r *RabbitMq) BindQueue(ch *amqp.Channel) error {
	return ch.QueueBind(
		r.Queue,    // queue name
		"",         // routing key (empty for direct exchange)
		r.Exchange, // exchange name
		false,      // no-wait
		nil,        // arguments
	)
}

func (r *RabbitMq) PublishMessage(ch *amqp.Channel, jsonMessage []byte) error {
	return ch.Publish(
		r.Exchange, // exchange
		"",         // routing key (queue name)
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonMessage, // jsonMessage is the marshaled struct
		},
	)
}

func (r *RabbitMq) ConsumeMessages(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	return ch.Consume(
		r.Queue, // queue name
		"",      // consumer tag
		false,   // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // arguments
	)
}
