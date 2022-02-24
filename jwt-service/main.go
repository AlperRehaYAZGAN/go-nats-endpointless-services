package main

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
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
	jwtSecret := os.Getenv("JWT_SECRET")
	// init nats connection
	InitNatsConnection(natsUrl)

	nc.Subscribe("jwt.validate", func(m *nats.Msg) {
		log.Printf("%s - Received long Jwt String. Jwt is validating...", time.Now().Format("2006-01-02 15:04:05"))
		// validate jwt
		token, err := jwt.Parse(string(m.Data), func(token *jwt.Token) (interface{}, error) {
			// validate secret
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil {
			nc.Publish(m.Reply, []byte("error-while-validating-jwt"))
			return
		}
		// validate token
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			nc.Publish(m.Reply, []byte(claims["data"].(string)))
		} else {
			nc.Publish(m.Reply, []byte("error-while-parsing-token"))
		}
	})

	nc.Subscribe("jwt.generate", func(m *nats.Msg) {
		log.Printf("%s - Received %s to generate Jwt. Jwt is generating...", time.Now().Format("2006-01-02 15:04:05"), m.Data)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"data": string(m.Data),
			"exp":  time.Now().Add(time.Hour * 24).Unix(),
		})
		// Create the JWT string
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			nc.Publish(m.Reply, []byte("error"))
		}
		nc.Publish(m.Reply, []byte(tokenString))
	})

	// log app started
	log.Printf("%s - Jwt Service started", time.Now().Format("2006-01-02 15:04:05"))
	// dont exit app until KeyBoardInterrupt
	select {}
}
