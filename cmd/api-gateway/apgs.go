package apigateway

import (
	chspb "chatbackend/chat-history-service/chatbackend/chathistoryservice"
	cspb "chatbackend/chat-service/chatbackend/chatservice"
	uspb "chatbackend/user-service/chatbackend/userservice"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type apigatewayServer struct {
	address    string
	csAddress  string
	csClient   cspb.ChatServiceClient
	usAddress  string
	usClient   uspb.RegisterClient
	chsAddress string
	chsClient  chspb.ChatHistoryServiceClient
}

type RegisterReq struct {
	UserName string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoadReq struct {
	Jwt string `json:"jwt"`
}

type ChatReq struct {
	UserMessage string `json:"userMessage"`
}

func NewServer(address string, csAddress string, usAddress string, chsAddress string) *apigatewayServer {
	return &apigatewayServer{
		address:    address,
		csAddress:  csAddress,
		usAddress:  usAddress,
		chsAddress: chsAddress,
	}
}
func (apiS *apigatewayServer) Start(ctx context.Context, errChan chan error) {
	// connect to chat history server

	creds := insecure.NewCredentials()
	chsConn, err := grpc.DialContext(ctx, apiS.chsAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		errChan <- err
	}
	apiS.chsClient = chspb.NewChatHistoryServiceClient(chsConn)

	go func() {

		<-ctx.Done()
		chsConn.Close()

	}()
	// connect to chat server

	csConn, err := grpc.DialContext(ctx, apiS.csAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		errChan <- err
	}
	apiS.csClient = cspb.NewChatServiceClient(csConn)
	go func() {
		<-ctx.Done()
		csConn.Close()

	}()

	// connect to user server

	usConn, err := grpc.DialContext(ctx, apiS.usAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		errChan <- err
	}
	apiS.usClient = uspb.NewRegisterClient(usConn)
	go func() {
		<-ctx.Done()
		usConn.Close()

	}()

	http.HandleFunc("/register", apiS.registerHandler)
	http.HandleFunc("/login", apiS.loginHandler)
	http.HandleFunc("/load", apiS.loadHandler)
	http.HandleFunc("/chat", apiS.chatHandler)

	go func() {

		err = http.ListenAndServe(apiS.address, nil)

		errChan <- err
	}()

}

func (apiS *apigatewayServer) registerHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Received register request")
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var data RegisterReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	register := uspb.RegisterRequest{
		Username: data.UserName,
		Email:    data.Email,
		Password: data.Password,
	}

	resp, err := apiS.usClient.Register(context.Background(), &register, grpc.WaitForReady(true))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfm := resp.GetRegisterCfm()
	if cfm != nil {
		respData := map[string]string{
			"jwt": cfm.GetJwt(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respData)
		return
	}

	rej := resp.GetRegisterRej()
	if rej != nil {
		respData := map[string]string{
			"error": rej.GetError(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respData)
		return
	}
}

func (apiS *apigatewayServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Received login request")
	var data LoginReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	login := uspb.LoginRequest{
		Email:    data.Email,
		Password: data.Password,
	}

	resp, err := apiS.usClient.Login(r.Context(), &login)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfm := resp.GetLoginCfm()
	if cfm != nil {
		respData := map[string]string{
			"jwt": cfm.GetJwt(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respData)
		return
	}

	rej := resp.GetLoginRej()
	if rej != nil {
		respData := map[string]string{
			"error": rej.GetError(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respData)
		return
	}
}

func (apiS *apigatewayServer) loadHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Received load request")
	var data LoadReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	loadReq := chspb.LoadChatHistoryReq{
		Jwt: data.Jwt,
	}

	resp, err := apiS.chsClient.LoadChatHistory(r.Context(), &loadReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfm := resp.GetLoadChatHistoryCfm()
	if cfm != nil {

		respData := map[string]string{
			"history": strings.Join(cfm.GetChatHistory(), ""),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respData)
		return
	}
	rej := resp.GetLoadChatHistoryRej()
	if rej != nil {
		respData := map[string]string{
			"error": rej.GetError(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respData)
		return
	}
}

func (apiS *apigatewayServer) chatHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Received chat request")
	var data ChatReq
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := cspb.ChatRequest{
		UserMessage: data.UserMessage,
	}

	stream, err := apiS.csClient.Chat(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			// End of stream
			break
		}
		if err != nil {
			http.Error(w, "Error reading from gRPC stream", http.StatusInternalServerError)
			return
		}

		// Write the streamed chunk to the HTTP response
		// For simplicity, assuming the streamed data is a string
		message := resp.GetAiResp()
		data := fmt.Sprintf("data: {\"message\": \"%s\"}\n\n", message)
		w.Write([]byte(data))
	}

}
