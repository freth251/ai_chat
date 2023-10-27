package chathistoryservice

import (
	chspb "chatbackend/chat-history-service/chatbackend/chathistoryservice"
	"chatbackend/common/env"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	serverRetryInterval = 2 * time.Second
	dbRetryInterval     = 2 * time.Second
)

type chathistoryserviceServer struct {
	chspb.UnimplementedChatHistoryServiceServer
	address string
	db      *sql.DB
	dbConn  string
}

func NewServer(address string, psqlInfo string) (*chathistoryserviceServer, error) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err) // TO-DO: proper log
		return nil, err
	}
	return &chathistoryserviceServer{
		address: address,
		db:      db,
		dbConn:  psqlInfo,
	}, nil
}

func (chS *chathistoryserviceServer) Start(ctx context.Context, errChan chan error) {
	err := chS.db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	logrus.Info("Succesfully connected to db")

	grpcServer := grpc.NewServer()
	chspb.RegisterChatHistoryServiceServer(grpcServer, &chathistoryserviceServer{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				logrus.Info("context cancelled")
				errChan <- context.Canceled

			default:
				lis, err := net.Listen("tcp", chS.address)
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

	//Monitor DB
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context cancelled") //TO DO: Should be a log
				errChan <- context.Canceled
				chS.db.Close()
			default:
				err := chS.db.Ping()

				if err != nil {
					errChan <- fmt.Errorf("db connection failed, reconnecting")
					db, err := sql.Open("postgres", chS.dbConn)
					if err != nil {
						errChan <- fmt.Errorf("reconnection failed, retrying in %v", dbRetryInterval)
						time.Sleep(dbRetryInterval)
						continue
					}
					chS.db = db
					time.Sleep(dbRetryInterval)
					continue

				}
				logrus.Info("Db connection up for chat history service")
				time.Sleep(dbRetryInterval)

			}
		}

	}()

}

func (chS *chathistoryserviceServer) LoadChatHistory(ctx context.Context, req *chspb.LoadChatHistoryReq) (*chspb.LoadChatHistoryResp, error) {
	if !validateLoadChatHistoryReq(req) {
		return retLoadChatHistoryRej(fmt.Errorf("invalid request")), nil
	}
	tokenString := req.GetJwt()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// This function will return the secret key.
		// In real-world applications, make sure to use a secure way to store and access it.
		return []byte(env.SecretKey), nil
	})

	if err != nil {
		return retLoadChatHistoryRej(fmt.Errorf("error decoding token: %v", err)), nil
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userId := claims["id"]
		var messages []string
		err = chS.db.QueryRow("SELECT messages FROM chat_history WHERE user_id=$1", userId).Scan(&messages)
		if err != nil {
			return retLoadChatHistoryRej(err), nil
		}
		cfm := chspb.LoadChatHistoryCfm{
			ChatHistory: messages,
		}
		resp := chspb.LoadChatHistoryResp_LoadChatHistoryCfm{
			LoadChatHistoryCfm: &cfm,
		}
		ret := chspb.LoadChatHistoryResp{
			LoadChatHistoryResp: &resp,
		}
		return &ret, nil
	} else {
		return retLoadChatHistoryRej(fmt.Errorf("invalid token or claims")), nil

	}

}

func (chS *chathistoryserviceServer) AddChatHistory(ctx context.Context, req *chspb.AddChatData) (*emptypb.Empty, error) {
	if !validateAddChatHistoryReq(req) {
		return &emptypb.Empty{}, nil

	}
	tokenString := req.GetJwt()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// This function will return the secret key.
		// In real-world applications, make sure to use a secure way to store and access it.
		return []byte(env.SecretKey), nil
	})

	if err != nil {
		return &emptypb.Empty{}, nil
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userId := claims["id"]
		chS.db.Exec("UPDATE chat_history SET messages = array_append(messages, $2) WHERE user_id=$1", userId, req.GetData())
	}
	return &emptypb.Empty{}, nil
}

func validateLoadChatHistoryReq(req *chspb.LoadChatHistoryReq) bool {
	return req.GetJwt() != ""
}
func validateAddChatHistoryReq(req *chspb.AddChatData) bool {
	if req.GetJwt() == "" || req.GetData() == "" {
		return false
	}
	return true
}

func retLoadChatHistoryRej(err error) *chspb.LoadChatHistoryResp {
	rej := chspb.LoadChatHistoryRej{
		Error: err.Error(),
	}
	resp := chspb.LoadChatHistoryResp_LoadChatHistoryRej{
		LoadChatHistoryRej: &rej,
	}

	ret := chspb.LoadChatHistoryResp{
		LoadChatHistoryResp: &resp,
	}
	return &ret
}
