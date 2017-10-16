package clickstream

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/astronomerio/event-router/kafka"
	"github.com/astronomerio/event-router/pkg/prom"
	cluster "github.com/bsm/sarama-cluster"
	confluent "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField("package", "clickstream")
)

type Consumer struct {
	options        *ConsumerOptions
	messageHandler kafka.MessageHandler
}

type ConsumerOptions struct {
	BootstrapServers string
	GroupID          string
	Topics           []string
	MessageHandler   kafka.MessageHandler
}

func NewConsumer(opts *ConsumerOptions) (*Consumer, error) {
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	return &Consumer{
		options:        opts,
		messageHandler: opts.MessageHandler,
	}, nil
}

func (c *Consumer) Run() {
	logger := log.WithField("function", "Run")
	logger.Info("Starting Kafka Consumer")
	logger.WithFields(logrus.Fields{
		"Bootstrap Servers": c.options.BootstrapServers,
		"GroupID":           c.options.GroupID,
		"Topics":            c.options.Topics,
	}).Debug("Consumer Options")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	consumer, err := confluent.NewConsumer(&confluent.ConfigMap{
		"bootstrap.servers":               c.options.BootstrapServers,
		"group.id":                        c.options.GroupID,
		"session.timeout.ms":              6000,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
		"enable.auto.commit":              true,
		"default.topic.config":            confluent.ConfigMap{"auto.offset.reset": "earliest"}})

	if err != nil {
		logger.Error(err)
		return
	}

	err = consumer.SubscribeTopics(c.options.Topics, nil)
	if err != nil {
		logger.Error(err)
		return
	}

	run := true
	for run == true {
		select {
		case sig := <-sigchan:
			logger.Infof("Consumer caught signal %v: terminating", sig)
			run = false

		case ev := <-consumer.Events():
			switch e := ev.(type) {
			case confluent.AssignedPartitions:
				logger.Infof("Assigning Partition %v", e)
				consumer.Assign(e.Partitions)
			case confluent.RevokedPartitions:
				logger.Infof("Revoking Partition %v", e)
				consumer.Unassign()
			case *confluent.Message:
				go c.messageHandler.HandleMessage(e.Value, e.Key)
				go prom.MessagesConsumed.Inc()
			case confluent.PartitionEOF:
				logger.Infof("Reached %v", e)
			case confluent.Error:
				logger.Error(e.Error())
			}
		}
	}

	consumer.Close()
	logger.Info("Consumer Closed")
}
