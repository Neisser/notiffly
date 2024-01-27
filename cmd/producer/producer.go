package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"notifly/pkg/models"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
)

const (
	ProducerPort       = ":8080"
	kafkaServerAddress = "localhost:9092"
	kafkaTopic         = "notifications"
)

// ============== HELPER FUNCTIONS ==============
var ErrUserNotFoundInProducer = errors.New("user not found in producer")

func findUserById(id int, users []models.User) (models.User, error) {
	for _, user := range users {
		if user.ID == id {
			return user, nil
		}
	}
	return models.User{}, ErrUserNotFoundInProducer
}

// getIdFromRequest retrieves an integer ID from the specified form value in the given Gin context.
// It prints the value of the form field to the console and attempts to convert it to an integer.
// If the conversion is successful, it returns the ID. Otherwise, it returns an error.
func getIdFromRequest(fromValue string, ctx *gin.Context) (int, error) {
	// Print the value of the form field to the console
	fmt.Println(ctx.PostForm(fromValue))

	// Convert the form field value to an integer
	id, err := strconv.Atoi(ctx.PostForm(fromValue))
	if err != nil {
		return 0, err
	}

	return id, nil
}

// ============== KAFKA RELATED FUNCTIONS ==============
func sendKafkaMessage(producter sarama.SyncProducer, users []models.User, ctx *gin.Context, fromId, toId int) error {
	message := ctx.PostForm("message")

	fromUser, err := findUserById(fromId, users)
	if err != nil {
		return err
	}

	toUser, err := findUserById(toId, users)
	if err != nil {
		return err
	}

	notification := models.Notification{
		From:    fromUser,
		To:      toUser,
		Message: message,
	}

	notificationJson, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("error marshalling notification: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: kafkaTopic,
		Key:   sarama.StringEncoder(strconv.Itoa(toUser.ID)),
		Value: sarama.StringEncoder(notificationJson),
	}

	_, _, err = producter.SendMessage(msg)
	return err
}

func SendMessageHandler(producer sarama.SyncProducer, users []models.User) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		fromId, err := getIdFromRequest("fromId", ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		toId, err := getIdFromRequest("toId", ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"Message": err.Error()})
			return
		}

		err = sendKafkaMessage(producer, users, ctx, fromId, toId)
		if errors.Is(err, ErrUserNotFoundInProducer) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		}
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "Notification sent successfully!"})
	}
}

func setupProducer() (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{kafkaServerAddress}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup producer: %w", err)
	}

	return producer, nil
}

func main() {
	users := []models.User{
		{ID: 1, Name: "Emma"},
		{ID: 2, Name: "Bruno"},
		{ID: 3, Name: "Rick"},
		{ID: 4, Name: "Lena"},
	}

	producer, err := setupProducer()
	if err != nil {
		log.Fatalf("failed to initialize producer: %v", err)
	}
	defer producer.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.POST("/send", SendMessageHandler(producer, users))

	fmt.Printf("Kafka PRODUCER 📨 started at http://localhost%s\n", ProducerPort)

	if err := router.Run(ProducerPort); err != nil {
		log.Fatalf("failed to run the server: %v", err)
	}
}
