package activemq

import (
	"github.com/streadway/amqp"
	"log"
)

type ActiveMQ struct {
	Addr string
}

func NewActiveMQ(addr string) *ActiveMQ {
	return &ActiveMQ{addr}
}

// Connect to activeMQ
func (mq *ActiveMQ) Connect() (*amqp.Connection, error) {
	return amqp.Dial(mq.Addr)
}

// Send msg to destination
func (mq *ActiveMQ) Send(msg []byte) error {
	conn, err := mq.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	q, err := ch.QueueDeclare(
		"publisher", // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)

	if err != nil {
		log.Fatalf("%s: %s", "Failed to declare a queue", err)
	}

	return ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		},
	)
}