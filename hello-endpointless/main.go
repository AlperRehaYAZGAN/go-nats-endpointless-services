package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats"
)

/**
*	ConnectNats : Connect to Nats
 */
var nc *nats.Conn

func InitNatsConnection(natsUrl string) {
	var natsErr error                   // natsUrl from .env » "nats://localhost:4222"
	nc, natsErr = nats.Connect(natsUrl) // connect to nats
	if natsErr != nil {
		log.Fatal("Fatal error happened while initial connection NATS »", natsErr)
	}
}

func main() {
	// current directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	// load .env file from path.join (process.cwd() + .env)
	err = godotenv.Load(dir + "/.env.test")
	if err != nil {
		// not found .env file. Log print not fatal
		log.Print("Error loading .env file ENV variables using if exist instead. ", err)
	}

	natsUrl := os.Getenv("NATS_URL")
	// init nats connection
	InitNatsConnection(natsUrl)

	r := gin.Default()

	r.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"hello": "World",
		})
	})

	r.GET("/encrypt/:message", func(ctx *gin.Context) {
		message := ctx.Param("message")
		// Nats request reply to encrypt message with 5 sec timeout
		ncReply, err := nc.Request("jwt.generate", []byte(message), time.Second*5)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"status":  false,
				"type":    "jwt-generate-from-service",
				"message": "Jwt Service Return Error. Check your inputs!",
				"error":   err.Error(),
			})
			return
		}
		// return jwt to user
		ctx.JSON(http.StatusOK, gin.H{
			"status":  true,
			"type":    "jwt-generate-from-jwt-service",
			"message": "Jwt Generated Successfully!",
			"jwt":     string(ncReply.Data),
		})

	})

	r.GET("/decrypt", func(ctx *gin.Context) {
		// get jwt from auth Bearer header
		jwtHeader := ctx.GetHeader("Authorization")
		if !(strings.HasPrefix(jwtHeader, "Bearer ")) {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"status":  false,
				"type":    "jwt-header-missing",
				"message": "Jwt is empty!",
			})
		}
		jwtHeader = strings.TrimPrefix(jwtHeader, "Bearer ")
		// Nats request reply to decrypt message with 5 sec timeout
		ncReply, err := nc.Request("jwt.validate", []byte(jwtHeader), time.Second*5)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"status":  false,
				"type":    "jwt-validate-from-jwt-service",
				"message": "Jwt Service Return Error. Check your inputs!",
				"error":   err.Error(),
			})
			return
		}

		// if ncReply.Data starts with "error" then it's an error
		if strings.HasPrefix(string(ncReply.Data), "error") {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"status":  false,
				"type":    "jwt-validate-from-jwt-service",
				"message": "Jwt is invalid!",
				"error":   string(ncReply.Data),
			})
			return
		} else {
			// return jwt to user
			ctx.JSON(http.StatusOK, gin.H{
				"status":  true,
				"type":    "jwt-validate-from-jwt-service",
				"message": "Jwt Validated Successfully!",
				"jwt":     string(ncReply.Data),
			})
		}
	})

	// start server on port APP_PORT
	APP_PORT := os.Getenv("APP_PORT")
	if APP_PORT == "" {
		APP_PORT = "9090"
	}
	if err := r.Run(":" + APP_PORT); err != nil {
		log.Fatal(err)
	}
}
