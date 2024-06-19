package main

import (
	"context"
	"fmt"
	"log"

	pb "dariche/internal/grpc"
	envsModule "dariche/pkg/envs"
	loggerModule "dariche/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	envs := envsModule.ReadEnvs()
	zapLogger := loggerModule.InitialLogger(envs.LOG_LEVEL)

	app := fiber.New()
	app.Use(logger.New())

	clientConn, err := grpc.Dial(
		fmt.Sprintf("%s:%s", envs.GRPC_SERVER_ADDRESS, envs.GRPC_SERVER_PORT),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to the gRPC server: %v \n", err)
	}
	defer clientConn.Close()

	grpcClient := pb.NewScrapperServiceClient(clientConn)

	app.Get("/api/v1/vulnerabilities/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")

		zapLogger.Info("Start searching for vulneabilities ...")
		res, err := grpcClient.FetchVulnerabilities(context.Background(), &pb.VulnerabilityRequest{
			Name: name,
		})
		if err != nil {
			fmt.Printf("Failed to call FetchVulnerabilities rpc: %v \n", err)
			return err
		}

		return c.JSON(res.Vulnerabilities)
	})

	app.Listen(":3040")
}
