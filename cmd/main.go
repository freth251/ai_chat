package main

import (
	apigateway "chatbackend/api-gateway"
	chs "chatbackend/chat-history-service"
	cs "chatbackend/chat-service"
	"chatbackend/common/env"
	"chatbackend/common/ports"
	ms "chatbackend/model-service"
	us "chatbackend/user-service"
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
)

const (
	hostname = "localhost"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	chAddress := fmt.Sprintf("%v:%d", hostname, ports.CsiPort)
	chsAddress := fmt.Sprintf("%v:%d", hostname, ports.ChsiPort)
	msAddress := fmt.Sprintf("%v:%d", hostname, ports.MsiPort)
	usAddress := fmt.Sprintf("%v:%d", hostname, ports.UsiPort)
	agAddress := fmt.Sprintf("%v:%d", hostname, ports.ApgsiPort)

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=disable",
		hostname, ports.DBPort, env.DBUserName, env.DBName,
	)
	chatHistoryServer, err := chs.NewServer(chsAddress, psqlInfo)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	userServer, err := us.NewServer(usAddress, psqlInfo)
	if err != nil {
		logrus.Error("error: ", err)
		return
	}
	chatServer := cs.NewServer(chAddress, chsAddress, msAddress)

	modelServer := ms.NewServer(msAddress)

	apigateway := apigateway.NewServer(agAddress, chAddress, usAddress, chsAddress)

	errChan := make(chan error)

	userServer.Start(ctx, errChan)
	modelServer.Start(ctx, errChan)
	chatHistoryServer.Start(ctx, errChan)
	chatServer.Start(ctx, errChan)
	go apigateway.Start(ctx, errChan)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled")
			return
		case err = <-errChan:
			fmt.Printf("error received in errChan: %v\n", err)
		}

	}

}
