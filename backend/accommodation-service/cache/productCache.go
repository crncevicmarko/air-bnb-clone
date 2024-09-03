package cache

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	// "log"
	"os"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

type ProductCache struct {
	cli    *redis.Client
	logger *log.Logger
}

// Construct Redis client
func New(logger *log.Logger) *ProductCache {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisAddress := fmt.Sprintf("%s:%s", redisHost, redisPort)

	client := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})

	return &ProductCache{
		cli:    client,
		logger: logger,
	}
}

func (pc *ProductCache) Ping() {
	val, _ := pc.cli.Ping(context.Background()).Result()
	pc.logger.Println(val)
}

func (pc *ProductCache) Post(image []byte, imageName string) error {
	log.Println("Usli u Post")
	key := strings.TrimSpace(imageName)

	// Encode the image data to base64
	encodedImage := base64.StdEncoding.EncodeToString(image)

	// Store the encoded image in Redis
	err := pc.cli.Set(context.Background(), constructKey(key), encodedImage, 30*time.Second).Err()
	if err != nil {
		log.Println("Error setting value in Redis:", err)
		return err
	}

	log.Println("Image successfully saved in Redis")
	return nil
}

func (pc *ProductCache) GetImage(key string) ([]byte, error) {
	encodedImage, err := pc.cli.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Println("Error in getting image from Redis or there is no one:", err)
			return nil, errors.New("cache miss")
		}
		log.Println("Error getting value from Redis:", err)
		return nil, err
	}

	// Decode base64-encoded image data
	decodedImage, err := base64.StdEncoding.DecodeString(encodedImage)
	if err != nil {
		log.Println("Error decoding image data:", err)
		return nil, err
	}

	log.Println("Cache hit")
	return decodedImage, nil
}

func (pc *ProductCache) Exists(id string) bool {
	cnt, err := pc.cli.Exists(context.Background(), constructKey(id)).Result()
	if cnt == 1 {
		return true
	}
	if err != nil {
		return false
	}
	return false
}
