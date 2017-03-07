package producer

import (
_"os"
"errors"
"github.com/Shopify/sarama"
"fmt"
)

var config *msgQConfig
var sProducer *msgQSyncProducer
var asProducer *msgQAsyncProducer

var syncMsgSent int32
var aSyncMsgEnqueued int32
var syncMsgErrCount int32
var aSyncMsgErrCount int32

type msgQConfig struct {
     msgQBrokers []string // array of strings like "1.2.3.4:9092" both ip and port are required
     msgQConfig *sarama.Config
     msgQAsyncConfig *sarama.Config
}

type msgQSyncProducer struct {
     producer sarama.SyncProducer
}

type msgQAsyncProducer struct {
     producer sarama.AsyncProducer
}

func Init(clientId string, msgQBrokers []string) error{
     if msgQBrokers == nil || len(msgQBrokers) <= 0 {
        return errors.New("Invalid Broker IP List")
     }
     config = &msgQConfig {}
     config.msgQBrokers = msgQBrokers
     config.msgQConfig = sarama.NewConfig()
     config.msgQConfig.Version = sarama.V0_9_0_1
     config.msgQConfig.ClientID = clientId 
     config.msgQConfig.Producer.Return.Successes = true
     config.msgQAsyncConfig = sarama.NewConfig()
     config.msgQAsyncConfig.Version = sarama.V0_9_0_1
     config.msgQAsyncConfig.ClientID = clientId
     config.msgQAsyncConfig.Producer.Return.Successes = false
     if  sProducer != nil {
         if sProducer.producer != nil {
            sProducer.producer.Close()
         }
     }
     if  asProducer != nil {
         if asProducer.producer != nil {
            asProducer.producer.Close()
         }
     }
     producer, err := sarama.NewSyncProducer(config.msgQBrokers, config.msgQConfig)
     if err != nil {
        fmt.Println("Error in initialization of SyncProducer:", err)
        return err
     }
     sProducer = &msgQSyncProducer {producer: producer}
     asproducer, err := sarama.NewAsyncProducer(config.msgQBrokers, config.msgQAsyncConfig)
     if err != nil {
        fmt.Println("Error in initialization of AsyncProducer:", err)
        return err
     }
     asProducer = &msgQAsyncProducer {producer: asproducer}
     doAsyncMsgAccounting(asProducer)
     return nil
}


func doAsyncMsgAccounting(aSyncProducer *msgQAsyncProducer) {
     go func() {
        for {
            select {
              case err := <-aSyncProducer.producer.Errors():
                   _ = err
                   aSyncMsgErrCount++
            }
        }
     }()
}


func PrintMsgCounters() {
     fmt.Println("SyncMsgSent: ",syncMsgSent, "SyncMsgErrorCnt: ",syncMsgErrCount)
     fmt.Println("AsyncMsgEnqueued: ",aSyncMsgEnqueued, "AsyncMsgErrorCnt: ",aSyncMsgErrCount)
}

func PrintConfig() {
     if config == nil {
        fmt.Println("Producer Config Not initialized")
        return
     }
     fmt.Printf("Config: brokers:%s  sync Config:%+v async Config\n",config.msgQBrokers,config.msgQAsyncConfig)
}

func SendSyncMessage(topic string, partitionHint string, val []byte) error {
     var encoder sarama.Encoder
     if partitionHint == "" {
        encoder = nil
     } else {
        encoder = sarama.StringEncoder(partitionHint)
     }

     msg := &sarama.ProducerMessage{
         Topic: topic,
         Key: encoder,
         Value: sarama.StringEncoder(val),
     }
     partition, offset, err := sProducer.producer.SendMessage(msg)
     _ = partition
     _ = offset
     if err != nil {
        syncMsgErrCount++
     } else {
        syncMsgSent++
     }
     return err
}

func SendAsyncMessage(topic string, partitionHint string, val []byte) {
     var encoder sarama.Encoder
     if partitionHint == "" {
        encoder = nil
     } else {
        encoder = sarama.StringEncoder(partitionHint)
     }

     msg := &sarama.ProducerMessage{
         Topic: topic,
         Key: encoder,
         Value: sarama.StringEncoder(val),
     }
     asProducer.producer.Input() <- msg
     aSyncMsgEnqueued++
}


