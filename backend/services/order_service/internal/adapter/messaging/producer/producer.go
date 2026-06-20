// WHAT: Kafka Producer interface + Sarama implementation สำหรับ order_service
// WHY interface แทน concrete struct โดยตรง?
//   - Dependency Inversion → OutboxProcessor depend on interface ไม่ใช่ Sarama
//   - Mock ได้ง่ายใน unit test (ไม่ต้องเชื่อม Kafka จริง)
//   - ถ้าย้ายจาก Kafka → messaging system อื่น แค่ implement interface นี้ใหม่
package producer

import (
	"fmt"
	"logs"

	"github.com/IBM/sarama"
)

// EventProducer คือ interface สำหรับส่ง event ไป message broker
type EventProducer interface {
	Send(topic string, key string, msg []byte, headers map[string]string) error
	Close() error
}

type saramaProducer struct {
	// WHY SyncProducer แทน AsyncProducer?
	//   - SyncProducer blocks จน Kafka ตอบกลับ → รู้ทันทีว่าส่งผ่านหรือไม่
	//   - Async ยากจัดการ error (channel-based) และ Outbox Pattern อยู่แล้วรับประกัน at-least-once
	//   - Sync + WaitForAll = stronger durability guarantee (leader + followers ack)
	producer sarama.SyncProducer
}

func NewSaramaProducer(brokers []string) (EventProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	// WHY WaitForAll แทน WaitForLocal?
	//   - WaitForLocal เร็วกว่า แต่ถ้า leader crash หลัง ack → event หายได้
	//   - Outbox Pattern ยอม latency เล็กน้อยเพื่อ durability
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &saramaProducer{producer: p}, nil
}

// Send ส่ง message ไป Kafka Topic พร้อม Headers
// WHY รับ map[string]string headers แทน []sarama.RecordHeader?
//   - ช่อน Sarama type ออกจาก caller (OutboxProcessor ไม่ต้องรู้จัก sarama.RecordHeader)
//   - ทำให้ mock EventProducer ง่ายขึ้น
func (p *saramaProducer) Send(topic string, key string, msg []byte, headers map[string]string) error {
	var saramaHeaders []sarama.RecordHeader
	for k, v := range headers {
		saramaHeaders = append(saramaHeaders, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	message := &sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.StringEncoder(key),
		Value:   sarama.ByteEncoder(msg),
		Headers: saramaHeaders,
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		return err
	}

	logs.Info(fmt.Sprintf("Published to %s partition=%d offset=%d key=%s", topic, partition, offset, key))
	return nil
}

func (p *saramaProducer) Close() error {
	return p.producer.Close()
}
