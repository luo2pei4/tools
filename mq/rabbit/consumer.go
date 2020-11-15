package rabbit

// 这段代码是从网上拷贝过来，我觉得写得不错，而且有针对实际应用场景的一些说明。具体的文章地址在下面可以参考
// 我对原来的代码做了一点小改动，把原文handle函数中的go程改为了线性执行
// 原文地址：https://ms2008.github.io/2019/06/16/golang-rabbitmq/

import (
	"time"
	"tools/log"

	"github.com/streadway/amqp"
)

// Consumer 消费者结构体
type Consumer struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	connNotify    chan *amqp.Error
	channelNotify chan *amqp.Error
	quit          chan struct{}
	addr          string
	exchange      string
	queue         string
	routingKey    string
	consumerTag   string
	autoDelete    bool
	handler       func([]byte) error
}

// NewConsumer 新建consumer
func NewConsumer(addr, exchange, queue string, autoDelete bool, handler func([]byte) error) *Consumer {
	c := &Consumer{
		addr:        addr,
		exchange:    exchange,
		queue:       queue,
		routingKey:  "",
		consumerTag: "consumer",
		autoDelete:  autoDelete,
		handler:     handler,
		quit:        make(chan struct{}),
	}

	return c
}

// Start 开始启动消费者端
func (c *Consumer) Start() error {

	log.Info("Start rabbitmq consumer.")

	if err := c.run(); err != nil {

		return err
	}

	go c.reConnect()

	return nil
}

// Stop 停止消费者端
func (c *Consumer) Stop() {

	close(c.quit)

	if !c.conn.IsClosed() {

		// 关闭 SubMsg message delivery
		if err := c.channel.Cancel(c.consumerTag, true); err != nil {

			log.Error("rabbitmq consumer - channel cancel failed: "+err.Error(), false)
		}

		if err := c.conn.Close(); err != nil {

			log.Error("rabbitmq consumer - connection close failed: "+err.Error(), false)
		}
	}
}

func (c *Consumer) run() error {

	log.Info("Run rabbitmq consumer.")

	var err error

	if c.conn, err = amqp.Dial(c.addr); err != nil {

		return err
	}

	if c.channel, err = c.conn.Channel(); err != nil {

		c.conn.Close()
		return err
	}

	if _, err = c.channel.QueueDeclare(

		c.queue,      // name
		false,        // durable
		c.autoDelete, // delete when usused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	); err != nil {

		c.channel.Close()
		c.conn.Close()
		return err
	}

	if err = c.channel.QueueBind(

		c.queue,
		c.routingKey,
		c.exchange,
		false,
		nil,
	); err != nil {

		c.channel.Close()
		c.conn.Close()
		return err
	}

	var delivery <-chan amqp.Delivery

	if delivery, err = c.channel.Consume(

		c.queue,       // queue
		c.consumerTag, // consumer
		false,         // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	); err != nil {

		c.channel.Close()
		c.conn.Close()
		return err
	}

	go c.handle(delivery)

	c.connNotify = c.conn.NotifyClose(make(chan *amqp.Error))
	c.channelNotify = c.channel.NotifyClose(make(chan *amqp.Error))

	return err
}

func (c *Consumer) reConnect() {

	for {

		select {

		case err := <-c.connNotify:

			if err != nil {

				log.Error("rabbitmq consumer - connection NotifyClose: "+err.Error(), false)
			}

		case err := <-c.channelNotify:

			if err != nil {

				log.Error("rabbitmq consumer - channel NotifyClose: "+err.Error(), false)
			}

		case <-c.quit:
			return
		}

		// backstop
		if !c.conn.IsClosed() {

			// close message delivery
			if err := c.channel.Cancel(c.consumerTag, true); err != nil {

				log.Error("rabbitmq consumer - channel cancel failed: "+err.Error(), false)
			}

			if err := c.conn.Close(); err != nil {

				log.Error("rabbitmq consumer - channel cancel failed: "+err.Error(), false)
			}
		}

		// 清空 Notify，否则死连接不会释放
		for err := range c.channelNotify {

			log.Error("rabbitmq consumer - release channel notify chan failed: "+err.Error(), false)
		}

		for err := range c.connNotify {

			log.Error("rabbitmq consumer - release connection notify chan failed: "+err.Error(), false)
		}

	quit:
		for {

			select {

			case <-c.quit:
				return

			default:

				log.Error("rabbitmq consumer - reconnect", false)

				if err := c.run(); err != nil {

					log.Error("rabbitmq consumer - failCheck: "+err.Error(), false)

					// sleep 5s reconnect
					time.Sleep(time.Second * 5)
					continue
				}

				break quit
			}
		}
	}
}

// Handle 消息处理函数
func (c *Consumer) handle(delivery <-chan amqp.Delivery) {

	for d := range delivery {

		if err := c.handler(d.Body); err == nil {

			d.Ack(false)

		} else {

			// 重新入队，否则未确认的消息会持续占用内存
			d.Reject(true)
		}
	}
}
