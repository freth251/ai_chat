package userservice

import (
	uspb "chatbackend/user-service/chatbackend/userservice"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/mail"
	"time"
	"unicode/utf8"

	"google.golang.org/grpc"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"chatbackend/common/env"
)

const (
	serverRetryInterval = 2 * time.Second
	dbRetryInterval     = 2 * time.Second
)

type userserviceServer struct {
	uspb.UnimplementedRegisterServer
	address string
	db      *sql.DB
	dbConn  string
}

type userInfo struct {
	userId   int
	userName string
	email    string
	password string
}

func NewServer(address string, psqlInfo string) (*userserviceServer, error) {
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return &userserviceServer{
		address: address,
		db:      db,
		dbConn:  psqlInfo,
	}, nil
}

func (uS *userserviceServer) Start(ctx context.Context, errChan chan error) {
	err := uS.db.Ping()
	if err != nil {
		logrus.Error("Initial ping to db failed: ", err)
		errChan <- err
		return
	}
	logrus.Info("Succesfully connected to db")

	grpcServer := grpc.NewServer()
	uspb.RegisterRegisterServer(grpcServer, uS)
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context cancelled") //TO DO: Should be a log
				errChan <- context.Canceled

			default:
				lis, err := net.Listen("tcp", uS.address)
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
				uS.db.Close()

			default:
				err := uS.db.Ping()

				if err != nil {
					errChan <- fmt.Errorf("db connection failed, reconnecting")
					db, err := sql.Open("postgres", uS.dbConn)
					if err != nil {
						errChan <- fmt.Errorf("reconnection failed, retrying in %v", dbRetryInterval)
						time.Sleep(dbRetryInterval)
						continue
					}
					uS.db = db
					time.Sleep(dbRetryInterval)
					continue

				}
				logrus.Info("Db connection up for user service")
				time.Sleep(dbRetryInterval)

			}
		}

	}()

}

func (uS *userserviceServer) Register(ctx context.Context, req *uspb.RegisterRequest) (*uspb.RegisterResp, error) {
	logrus.Info("Received register request : ", req)
	if !validateRegisterReq(req.GetUsername(), req.GetEmail(), req.GetPassword()) {
		return retRegisterRej(fmt.Errorf("invalid request")), nil
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return retRegisterRej(err), nil
	}

	// Insert into users table
	user := userInfo{}
	if uS.db == nil {
		logrus.Info("db is nill")
	}
	err = uS.db.QueryRow(`INSERT INTO users (username, email, password, hashsalt) VALUES ($1, $2, $3, $4) RETURNING id`, req.GetUsername(), req.GetEmail(), req.GetPassword(), string(hashedPassword)).Scan(&user.userId)
	if err != nil {
		logrus.Error(err)
		return retRegisterRej(err), nil
	}

	// Insert into chat_history table
	_, err = uS.db.Exec(`INSERT INTO chat_history (user_id) VALUES ($1)`, user.userId)
	if err != nil {
		logrus.Error(err)
		return retRegisterRej(err), nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.GetUsername(),
		"id":       user.userId,
		"email":    req.GetEmail(),
		"password": req.GetPassword(),
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	})

	tokenString, err := token.SignedString([]byte(env.SecretKey))
	if err != nil {
		return retRegisterRej(err), nil
	}

	cfm := uspb.RegisterCfm{
		Jwt: tokenString,
	}

	resp := uspb.RegisterResp_RegisterCfm{
		RegisterCfm: &cfm,
	}

	ret := uspb.RegisterResp{
		RegisterResp: &resp,
	}

	return &ret, nil
}

func (uS *userserviceServer) Login(ctx context.Context, req *uspb.LoginRequest) (*uspb.LoginResp, error) {
	logrus.Info("Received login request : ", req)
	if !validateLoginReq(req.GetEmail(), req.GetPassword()) {
		return retLoginRej(fmt.Errorf("invalid request")), nil
	}
	user := &userInfo{}
	err := uS.db.QueryRow("SELECT id, email, password, username FROM users WHERE email = $1", req.GetEmail()).Scan(&user.userId, &user.email, &user.password, &user.userName)
	if err != nil {
		return retLoginRej(err), nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.password), []byte(req.GetPassword()))
	if err != nil {
		return retLoginRej(err), nil
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.userName,
		"id":       user.userId,
		"email":    user.email,
		"password": user.password,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	})

	tokenString, err := token.SignedString([]byte(env.SecretKey))
	if err != nil {
		return retLoginRej(err), nil
	}

	cfm := uspb.LoginCfm{
		Jwt: tokenString,
	}

	resp := uspb.LoginResp_LoginCfm{
		LoginCfm: &cfm,
	}

	ret := uspb.LoginResp{
		LoginResp: &resp,
	}

	return &ret, nil
}

func validateRegisterReq(username, email, password string) bool {

	if len(username) < 3 || len(username) > 50 {
		return false
	}

	// Check if email is valid
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	if utf8.RuneCountInString(password) < 6 {
		return false
	}

	return true
}

func validateLoginReq(email, password string) bool {
	// Check if email is valid
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	if utf8.RuneCountInString(password) < 6 {
		return false
	}

	return true
}

func retRegisterRej(err error) *uspb.RegisterResp {
	rej := uspb.RegisterRej{
		Error: err.Error(),
	}

	resp := uspb.RegisterResp_RegisterRej{
		RegisterRej: &rej,
	}

	ret := uspb.RegisterResp{
		RegisterResp: &resp,
	}

	return &ret
}

func retLoginRej(err error) *uspb.LoginResp {
	rej := uspb.LoginRej{
		Error: err.Error(),
	}

	resp := uspb.LoginResp_LoginRej{
		LoginRej: &rej,
	}

	ret := uspb.LoginResp{
		LoginResp: &resp,
	}

	return &ret
}
