package pkg

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/IBM/sarama"
)

var kafkaClient sarama.Client
var kafkaProducer sarama.SyncProducer

// InitKafka initializes the shared Kafka client and sync producer.
// Brokers can be provided via argument; if empty, it reads KAFKA_BROKERS env (comma-separated).
func InitKafka(brokers []string) error {
	if kafkaProducer != nil {
		return nil
	}
	if len(brokers) == 0 {
		if v := os.Getenv("KAFKA_BROKERS"); v != "" {
			brokers = strings.Split(v, ",")
		}
	}
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_5_0_0
	cfg.Producer.Return.Successes = true
	cfg.Producer.Idempotent = true
	cfg.Net.MaxOpenRequests = 1

	c, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return err
	}
	p, err := sarama.NewSyncProducerFromClient(c)
	if err != nil {
		_ = c.Close()
		return err
	}
	kafkaClient = c
	kafkaProducer = p
	return nil
}

func CloseKafka() {
	if kafkaProducer != nil {
		_ = kafkaProducer.Close()
	}
	if kafkaClient != nil {
		_ = kafkaClient.Close()
	}
}

func PublishMessage(topic string, key, value []byte) error {
	msg := &sarama.ProducerMessage{Topic: topic}
	if key != nil {
		msg.Key = sarama.ByteEncoder(key)
	}
	msg.Value = sarama.ByteEncoder(value)
	_, _, err := kafkaProducer.SendMessage(msg)
	return err
}

func PublishJSON(topic string, key []byte, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return PublishMessage(topic, key, b)
}

// StartConsumerGroup starts a consumer group loop until ctx is cancelled.
func StartConsumerGroup(ctx context.Context, group string, topics []string, handler func(*sarama.ConsumerMessage) error) error {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_5_0_0
	cfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	if kafkaClient == nil {
		return sarama.ErrClosedClient
	}
	cg, err := sarama.NewConsumerGroupFromClient(group, kafkaClient)
	if err != nil {
		return err
	}

	go func() {
		defer cg.Close()
		for {
			if err := cg.Consume(ctx, topics, consumerGroupHandler{fn: handler}); err != nil {
				// loop until ctx cancelled; errors will break and retry
				if ctx.Err() != nil {
					return
				}
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return nil
}

type consumerGroupHandler struct {
	fn func(*sarama.ConsumerMessage) error
}

func (h consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		_ = h.fn(msg)
		sess.MarkMessage(msg, "")
	}
	return nil
}

// removed custom getenv; use os.Getenv directly
