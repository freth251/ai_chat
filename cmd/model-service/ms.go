package modelservice

import (
	"bufio"
	"bytes"
	"chatbackend/common/ports"
	mspb "chatbackend/model-service/chatbackend/modelservice"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

const (
	serverRetryInterval = 2 * time.Second
)

type modelserviceServer struct {
	mspb.UnimplementedModelServiceServer
	address string
}
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

func NewServer(address string) *modelserviceServer {
	return &modelserviceServer{
		address: address,
	}
}

func (mS *modelserviceServer) Start(ctx context.Context, errChan chan error) {

	grpcServer := grpc.NewServer()
	mspb.RegisterModelServiceServer(grpcServer, &modelserviceServer{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context cancelled") //TO DO: Should be a log
				errChan <- context.Canceled

			default:
				lis, err := net.Listen("tcp", mS.address)
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
}

func (mS *modelserviceServer) Model(req *mspb.ModelInput, stream mspb.ModelService_ModelServer) error {
	data := GenerateRequest{
		Model:  req.GetModel(),
		Prompt: req.GetPrompt(),
	}

	// Convert the struct to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		//TO DO: Add log
		return err
	}

	// Make the POST request
	apiCall := fmt.Sprintf("localhost:%v/api/generate", ports.ModelPort)
	resp, err := http.Post(apiCall, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		//TO DO: Add log
		return err
	}
	defer resp.Body.Close()

	// Read the streamed response
	reader := bufio.NewReader(resp.Body)
	for {
		b, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading response:", err)
			return err
		}
		resp := mspb.ModelOutput{
			Response: string(b),
		}
		if err := stream.Send(&resp); err != nil {
			return err
		}
	}
	return nil
}
