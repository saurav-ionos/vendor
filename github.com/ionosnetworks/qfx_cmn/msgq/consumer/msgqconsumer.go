package consumer

import (
"os"
"errors"
"github.com/Shopify/sarama"
"github.com/bsm/sarama-cluster"
"fmt"
)

var config *cluster.Config
type msgQConfig struct {
     msgQBrokers *[]string // array of strings like "1.2.3.4:9092" both ip and port are required
     msgQConfig *sarama.Config
     groupIdTopicMap *map[string][]string
}

var initConfig *msgQConfig


type MsgCallBack func(topic string, partition int32, msg []byte) 

func Init(clientId string, msgQBrokers []string, groupIdTopicMap map[string][]string, ch chan[]byte ) error {
     if msgQBrokers == nil || len(msgQBrokers) <= 0 {
        return errors.New("Invalid Broker IP List")
     }
     if groupIdTopicMap == nil || len(groupIdTopicMap) <= 0 {
        return errors.New("Invalid GroupId to Topic Map")
     }
     /*
     if msgCallback == nil {
        return errors.New("No Msg Call Back function provided")
     }*/
     
     config = cluster.NewConfig() 
     config.Config.Version = sarama.V0_9_0_1
     config.Config.ClientID = clientId 
     config.Consumer.Return.Errors = true
     config.Group.Return.Notifications = true
     initConfig = &msgQConfig{msgQBrokers: &msgQBrokers, msgQConfig: &config.Config, groupIdTopicMap:&groupIdTopicMap}
     for groupId,topics := range groupIdTopicMap {
             consumer, err := cluster.NewConsumer(msgQBrokers, groupId, topics, config)
         if err != nil {
            fmt.Println("Failed to start consumer: %s", err)
            os.Exit(1)
         }

         go func() {
            for err := range consumer.Errors() {
                fmt.Printf("Error: %s\n", err.Error())
            }
         }()

         go func() {
             for note := range consumer.Notifications() {
                 fmt.Printf("Rebalanced: %+v\n", note)
             }
         }()

         go func() {
             for msg := range consumer.Messages() {
                 //msgCallback(msg.Topic, msg.Partition, msg.Value)
                 ch<-msg.Value
                 consumer.MarkOffset(msg, "")
             }
         }()

     }
     return nil
}

func PrintConfig() {
     if initConfig == nil {
        fmt.Println("Consumer Config Not initialized")
        return
     }
     fmt.Printf("Config: brokers:%s sarama.Config:%+v \n", initConfig.msgQBrokers, initConfig.msgQConfig)
     fmt.Println("Subscription Config: ",initConfig.groupIdTopicMap)
}
