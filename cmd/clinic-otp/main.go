package main

import (
	"log"
	"net/http"
	"os"

	clinicotp "example.com/clinic-otp-gateway"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}

	handler := clinicotp.NewLoginHandler(clinicotp.NewSMSClient(apiKey)).Routes()
	log.Printf("clinic OTP gateway listening on %s", address)
	log.Fatal(http.ListenAndServe(address, handler))
}
