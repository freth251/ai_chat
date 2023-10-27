package chatservice

import (
	chspb "chatbackend/chat-history-service/chatbackend/chathistoryservice"
	cspb "chatbackend/chat-service/chatbackend/chatservice"
	"chatbackend/common/env"
	mspb "chatbackend/model-service/chatbackend/modelservice"
	"io"
	"log"

	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverRetryInterval = 2 * time.Second
)

type chatserviceServer struct {
	cspb.UnimplementedChatServiceServer
	address    string
	chsAddress string
	chsClient  chspb.ChatHistoryServiceClient
	msAddress  string
	msClient   mspb.ModelServiceClient
}

func NewServer(address string, chsAddress string, msAddress string) *chatserviceServer {
	return &chatserviceServer{
		address:    address,
		chsAddress: chsAddress,
		msAddress:  msAddress,
	}
}

func (cS *chatserviceServer) Start(ctx context.Context, errChan chan error) {
	grpcServer := grpc.NewServer()
	cspb.RegisterChatServiceServer(grpcServer, &chatserviceServer{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context cancelled") //TO DO: Should be a log
				errChan <- context.Canceled

			default:
				lis, err := net.Listen("tcp", cS.address)
				if err != nil {
					errChan <- fmt.Errorf("failed to listen: %v", err)
					time.Sleep(serverRetryInterval)
					continue
				}
				if err := grpcServer.Serve(lis); err != nil {
					errChan <- fmt.Errorf("failed to serve: %v", err)
					time.Sleep(serverRetryInterval)
					continue
				}

			}
		}

	}()

	// connect to chat history server

	creds := insecure.NewCredentials()
	chsConn, err := grpc.DialContext(ctx, cS.chsAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		errChan <- err
	}
	cS.chsClient = chspb.NewChatHistoryServiceClient(chsConn)

	go func() {

		<-ctx.Done()
		chsConn.Close()

	}()
	// connect to model server

	msConn, err := grpc.DialContext(ctx, cS.msAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		errChan <- err
	}
	cS.msClient = mspb.NewModelServiceClient(msConn)
	go func() {
		<-ctx.Done()
		msConn.Close()

	}()
}

func (cS *chatserviceServer) Chat(req *cspb.ChatRequest, stream cspb.ChatService_ChatServer) error {
	if !validateChatRequest(req) {
		return fmt.Errorf("invalid request")
	}

	msReq := mspb.ModelInput{
		Prompt: req.GetUserMessage(),
		Model:  env.DefaultModel,
	}

	streamMs, err := cS.msClient.Model(context.Background(), &msReq, grpc.WaitForReady(true))
	if err != nil {
		//TO DO: Add logs
		return err
	}

	userMsg := "USER:\t" + req.GetUserMessage()
	aiMsg := "AI:\t"
	for {
		msg, err := streamMs.Recv()

		if err == io.EOF {
			// end of the stream
			break
		}
		if err != nil {
			log.Fatalf("Error while reading stream: %v", err)
		}
		resp := cspb.ChatResp{
			AiResp: msg.GetResponse(),
		}
		if err := stream.Send(&resp); err != nil {
			return err
		}
		aiMsg = aiMsg + msg.GetResponse()
	}

	histMsg := userMsg + "\n" + aiMsg + "\n"

	chatHistoryReq := chspb.AddChatData{
		Jwt:  req.GetJwt(),
		Data: histMsg,
	}

	_, err = cS.chsClient.AddChatHistory(context.Background(), &chatHistoryReq, grpc.WaitForReady(true))
	if err != nil {
		//TO DO: Add logs
		return err
	}

	return nil
}

func validateChatRequest(req *cspb.ChatRequest) bool {
	if req.GetJwt() == "" || req.GetUserMessage() == "" {
		return false
	}
	return true
}
