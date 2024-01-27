package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"notifly/pkg/models"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
)

const (
	ConsumerGroup      = "notifications-group"
	ConsumerTopic      = "notifications"
	ConsumerPort       = ":8081"
	kafkaServerAddress = "localhost:9092"
)

// ============== HELPER FUNCTIONS ==============
var ErrNoMessagesFound = errors.New("no messages found")

func getUserIdFromRequest(ctx *gin.Context) (string, error) {
	userId := ctx.Param("userId")
	if userId == "" {
		return "", ErrNoMessagesFound
	}
	return userId, nil
}

// ============== NOTIFICATION STORAGE ==============
type UserNotifications map[string][]models.Notification

type NotificationStore struct {
	data UserNotifications
	mu   sync.RWMutex
}

func (ns *NotificationStore) Add(userId string, notification models.Notification) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	_, ok := ns.data[userId]
	if !ok {
		ns.data[userId] = []models.Notification{}
	}
	ns.data[userId] = append(ns.data[userId], notification)
}

func (ns *NotificationStore) Get(userId string) ([]models.Notification, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	notifications, ok := ns.data[userId]
	if !ok {
		return nil, ErrNoMessagesFound
	}
	return notifications, nil
}

// ============== KAFKA RELATED FUNCTIONS ==============
type Consumer struct {
	store *NotificationStore
}

func (*Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (*Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		userId := string(message.Key)
		var notification models.Notification

		err := json.Unmarshal(message.Value, &notification)
		if err != nil {
			log.Printf("failed to unmarshal notification: %v", err)
			continue
		}

		c.store.Add(userId, notification)

		session.MarkMessage(message, "")
	}
	return nil
}

func initializeConsumerGroup() (sarama.ConsumerGroup, error) {
	config := sarama.NewConfig()

	config.Version = sarama.V2_5_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGroup, err := sarama.NewConsumerGroup([]string{kafkaServerAddress}, ConsumerGroup, config)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer group: %w", err)
	}
	return consumerGroup, nil
}

func setupConsumerGroup(ctx context.Context, store *NotificationStore) error {
	consumerGroup, err := initializeConsumerGroup()
	if err != nil {
		return err
	}

	defer consumerGroup.Close()

	consumer := &Consumer{store: store}

	for {
		err := consumerGroup.Consume(ctx, []string{ConsumerTopic}, consumer)
		if err != nil {
			return fmt.Errorf("error from consumer group: %w", err)
		}

		if ctx.Err() != nil {
			return nil
		}
	}
}

// DONE: add documentation
func handleNotifications(ctx *gin.Context, store *NotificationStore) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}

	notes, err := store.Get(userId)
	// print notes len
	fmt.Println("notification len ", len(notes))
	if len(notes) == 0 {
		ctx.JSON(
			http.StatusOK,
			gin.H{
				"message":       "No notifications found",
				"notifications": []models.Notification{},
			},
		)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"notifications": notes})
}

func main() {
	store := &NotificationStore{
		data: make(UserNotifications),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go setupConsumerGroup(ctx, store)
	defer cancel()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/notifications/:userId", func(ctx *gin.Context) {
		handleNotifications(ctx, store)
	})

	fmt.Printf("Kafka CONSUMER (Group: %s) 👥📥 "+
		"started at http://localhost%s\n", ConsumerGroup, ConsumerPort)

	if err := router.Run(ConsumerPort); err != nil {
		log.Fatalf("failed to run the server: %v", err)
	}
}
