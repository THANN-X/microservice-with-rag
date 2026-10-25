package messaging

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/IBM/sarama"
)

// Header ที่แนบไปกับ message ใน DLQ เพื่อให้ย้อนกลับไปหาต้นทางและสาเหตุได้
// header เดิมของ message (EventType, EventID) ถูก copy ไปด้วยเสมอ → replay กลับ topic เดิมได้ทันที
const (
	HeaderOriginalTopic     = "dlq-original-topic"
	HeaderOriginalPartition = "dlq-original-partition"
	HeaderOriginalOffset    = "dlq-original-offset"
	HeaderConsumer          = "dlq-consumer"
	HeaderError             = "dlq-error"
	HeaderAttempts          = "dlq-attempts"
	HeaderFailedAt          = "dlq-failed-at"
)

// DeadLetterTopic ตั้งชื่อ DLQ จาก topic ต้นทาง เช่น order.events → order.events.dlq
// WHY แยก DLQ ต่อ topic? → รู้ทันทีว่า event ชนิดไหนพัง และ replay กลับ topic เดิมได้ตรงตัว
func DeadLetterTopic(topic string) string {
	return topic + ".dlq"
}

// DeadLetterPublisher ส่ง message ที่ประมวลผลไม่สำเร็จไปเก็บไว้ใน DLQ
type DeadLetterPublisher interface {
	Publish(ctx context.Context, consumer string, msg *sarama.ConsumerMessage, cause error, attempts int) error
}

// SaramaDeadLetterPublisher ใช้ SyncProducer เพราะต้องรู้ให้แน่ว่าเขียนลง DLQ สำเร็จ
// ก่อนจะ MarkMessage ต้นทาง ไม่งั้น message หายได้ระหว่างทาง
type SaramaDeadLetterPublisher struct {
	producer sarama.SyncProducer
}

func NewDeadLetterPublisher(brokers []string) (*SaramaDeadLetterPublisher, error) {
	cfg := sarama.NewConfig()
	cfg.Version = KafkaVersion
	cfg.Producer.RequiredAcks = sarama.WaitForAll // WHY: DLQ คือที่พักสุดท้ายของ message ห้ามหาย
	cfg.Producer.Return.Successes = true          // SyncProducer บังคับ
	cfg.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("create dead letter producer: %w", err)
	}
	return &SaramaDeadLetterPublisher{producer: producer}, nil
}

func (p *SaramaDeadLetterPublisher) Publish(_ context.Context, consumer string, msg *sarama.ConsumerMessage, cause error, attempts int) error {
	_, _, err := p.producer.SendMessage(buildDeadLetterMessage(consumer, msg, cause, attempts, time.Now()))
	return err
}

func (p *SaramaDeadLetterPublisher) Close() error {
	return p.producer.Close()
}

func buildDeadLetterMessage(consumer string, msg *sarama.ConsumerMessage, cause error, attempts int, failedAt time.Time) *sarama.ProducerMessage {
	headers := make([]sarama.RecordHeader, 0, len(msg.Headers)+7)
	for _, h := range msg.Headers {
		if h != nil {
			headers = append(headers, *h)
		}
	}
	headers = append(headers,
		header(HeaderOriginalTopic, msg.Topic),
		header(HeaderOriginalPartition, strconv.Itoa(int(msg.Partition))),
		header(HeaderOriginalOffset, strconv.FormatInt(msg.Offset, 10)),
		header(HeaderConsumer, consumer),
		header(HeaderError, cause.Error()),
		header(HeaderAttempts, strconv.Itoa(attempts)),
		header(HeaderFailedAt, failedAt.UTC().Format(time.RFC3339)),
	)

	out := &sarama.ProducerMessage{
		Topic:   DeadLetterTopic(msg.Topic),
		Value:   sarama.ByteEncoder(msg.Value),
		Headers: headers,
	}
	// WHY เช็คก่อน? ByteEncoder(nil) ไม่ใช่ nil interface → hash partitioner ส่งทุกตัวไป partition เดียว
	if len(msg.Key) > 0 {
		out.Key = sarama.ByteEncoder(msg.Key)
	}
	return out
}

func header(key, value string) sarama.RecordHeader {
	return sarama.RecordHeader{Key: []byte(key), Value: []byte(value)}
}
