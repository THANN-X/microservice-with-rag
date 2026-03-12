package producer

import (
	"fmt"
	"logs"

	"github.com/IBM/sarama"
)

// EventProducer เป็น interface โดยเจตนา (Dependency Inversion)
// WHY interface แทนแค่ struct?
//   - ให้ Test mock ได้ (MockProducer stub) โดยไม่ต้องเชื่อม Kafka จริง
//   - TODO: ถ้าย้ายจาก Kafka ไปใช้ Pub/Sub อื่น แค่ implement interface นี้ใหม่โดยไม่ต้องแก้ worker/service
type EventProducer interface {
	Send(topic string, key string, msg []byte, headers map[string]string) error
	Close() error
}

type saramaProducer struct {
	// SyncProducer blocks จน Kafka ตอบกลับ → รู้ทันทีว่าส่งผ่านหรือไม่
	// WHY ไม่ใช้ AsyncProducer?
	//   - Async ยากจัดการ error (channel-based) และ Outbox Pattern อยู่แล้ว
	//   - Sync Producer สอดคล้องกับ Outbox workflow: รู้ผลใน call เดียว mark SENT/FAILED
	producer sarama.SyncProducer // ใช้ Sync เพื่อความชัวร์ (ส่งไม่ผ่านจะได้รู้ทันที)
}

func NewSaramaProducer(brokers []string) (EventProducer, error) {
	config := sarama.NewConfig()
	// Producer.Return.Successes = true คือต้องการ เพราะใช้ SyncProducer
	config.Producer.Return.Successes = true
	// WaitForAll: รอให้ Kafka leader + followers ตอบกลับ (strongest durability guarantee)
	// WHY ไม่ใช้ WaitForLocal?
	//   - WaitForLocal เร็วกว่า แต่ถ้า leader crash หลัง ack → event หายได้
	//   - Outbox Pattern ยอม delay นิดหน่อยเพื่อความน่าเชื่อถือมากขึ้น
	config.Producer.RequiredAcks = sarama.WaitForAll // รอให้ Kafka ตอบกลับว่า save ลง disk ครบทุกเครื่อง (ช้าหน่อยแต่ชัวร์สุด)
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &saramaProducer{producer: producer}, nil
}

func (p *saramaProducer) Send(topic string, key string, msg []byte, headers map[string]string) error {
	// แปลง map[string]string → []sarama.RecordHeader
	// WHY ใช้ map input แทนที่จะรับ []RecordHeader ตรงๆ?
	//   - ชั้น caller (OutboxProcessor) ไม่ต้องรู้จัก sarama type
	//   - ซ่อน infrastructure detail ไว้ใน layer นี้
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
		logs.Error(fmt.Sprintf("Failed to send message to topic %s: %v", topic, err))
		return err
	}

	logs.Info(fmt.Sprintf("Message sent to topic %s [partition: %d, offset: %d]", topic, partition, offset))
	return nil
}

func (p *saramaProducer) Close() error {
	return p.producer.Close()
}
