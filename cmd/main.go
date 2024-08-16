package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	pb "dariche/internal/grpc"
	envsModule "dariche/pkg/envs"
	loggerModule "dariche/pkg/logger"
	redisModule "dariche/pkg/redis"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	envs := envsModule.ReadEnvs()
	zapLogger := loggerModule.InitialLogger(envs.LOG_LEVEL)
	redisClient := redisModule.Init(envs)

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

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		vulerabilities, err := redisClient.Get(ctx, name).Result()
		if err != nil {
			fmt.Printf("Cache Miss for %s\n", name)
		} else {
			fmt.Printf("Cache Hit for %s\n", name)
			return c.JSON(vulerabilities)
		}

		zapLogger.Info("Start searching for vulneabilities ...")
		res, err := grpcClient.FetchVulnerabilities(context.Background(), &pb.VulnerabilityRequest{
			Name: name,
		})
		if err != nil {
			fmt.Printf("Failed to call FetchVulnerabilities rpc: %v \n", err)
			return err
		}

		fmt.Println(len(res.Vulnerabilities))

		if len(res.Vulnerabilities) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)

			val, err := json.Marshal(res.Vulnerabilities)
			if err != nil {
				fmt.Println("Failed to insert in cache")
			} else {
				redisClient.Set(ctx, name, val, 72*time.Hour)
				fmt.Printf("vulnerabilities are inserted in cache for %s\n", name)
			}

			cancel()
		}

		return c.JSON(res.Vulnerabilities)
	})
	// app.Post("api/v1/fetch-vulnerabilities/:type", r.Handler.VulnerabilityHandler())
	// app.Get("api/v1/vulnerabilities/detail/:piplineId", r.Handler.FetchVulnerabilitiesDetails())

	app.Listen(":3040")
}
