package main

import (
	"context"
	"encoding/json"
	"errors"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgin "github.com/rookie-ninja/rk-gin/v2/boot"
	kafkago "github.com/segmentio/kafka-go"
	"humanVerification/internal/adapters/kafka"
	"humanVerification/internal/adapters/repository/postgres/human_verification"
	resthandlers "humanVerification/internal/adapters/rest"
	"humanVerification/internal/adapters/s3"
	"humanVerification/internal/core/service"
	"log"
	"os"
	"strings"
)

var responsesProducer *kafka.Producer
var verificationService *service.Service

// @title Human Verification Service
// @version 1.0
func main() {
	raw, err := os.ReadFile("boot.yaml")
	if err != nil {
		log.Fatalf("failed to read boot.yaml: %v", err)
	}

	boot := rkboot.NewBoot(rkboot.WithBootConfigRaw([]byte(os.ExpandEnv(string(raw)))))

	// Bootstrap
	boot.Bootstrap(context.TODO())

	minioEndpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "admin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "password123")
	minioBucket := getEnv("MINIO_BUCKET", "app-builds")
	minioPublicEndpoint := getEnv("MINIO_PUBLIC_ENDPOINT", minioEndpoint)
	minioRegion := getEnv("MINIO_REGION", "us-east-1")
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9092")

	gin := rkgin.GetGinEntry("human_verification")

	pgEntry := rkpostgres.GetPostgresEntry("human_verification_postgres")
	if pgEntry == nil {
		log.Fatal("postgres entry not found")
	}

	dbEntry := pgEntry.GetDB("human_verification")
	db, err := dbEntry.DB()
	if err != nil {
		panic(err)
	}

	responsesProducer = kafka.NewProducer([]string{kafkaBroker})
	defer func() {
		if err := responsesProducer.Close(); err != nil {
			log.Printf("producer close error: %v", err)
		}
	}()

	repository := human_verification.New(db)
	minioClient := s3.NewMinio(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioPublicEndpoint, minioRegion)
	verificationService = service.NewService(repository, responsesProducer, minioClient)
	handler := resthandlers.NewVerificationRequestHandler(verificationService)

	gin.Router.GET("/verifications", handler.GetRequestsListByStatuses)
	gin.Router.PATCH("/verifications/:requestId", handler.UpdateRequestStatus)
	gin.Router.GET("/verifications/:requestId/link", handler.GetRequestFileLink)

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: []string{kafkaBroker},
		GroupID: "virus.verify.request",
		Topic:   kafka.TopicHumanVerifyRequest,
	}, handleVerificationRequest)

	ctx, stop := context.WithCancel(context.Background())

	go func() {
		if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("kafka responses consumer stopped with error: %v", err)
		}

		defer stop()
	}()

	boot.WaitForShutdownSig(context.TODO())
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return fallback
}

func handleVerificationRequest(ctx context.Context, msg kafkago.Message) error {
	var req kafka.Event
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		return err
	}

	log.Printf("verification request accepted: %v", msg)

	if err := verificationService.CreateVerificationRequest(req); err != nil {
		log.Printf("verification request was not saved for %s: %v", req.CorrelationID, err)
	}

	log.Printf("verification request saved: %v", msg)
	return nil
}
