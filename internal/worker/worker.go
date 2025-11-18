package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	clientpkg "github.com/stormhead-org/backend/internal/client"
	eventpkg "github.com/stormhead-org/backend/internal/event"
	templatepkg "github.com/stormhead-org/backend/internal/template"
	ormpkg "github.com/stormhead-org/backend/internal/orm"
)

type Worker struct {
	context      context.Context
	cancel       func()
	waitGroup    sync.WaitGroup
	logger       *zap.Logger
	router       *Router
	brokerClient *eventpkg.KafkaClient
	mailClient   *clientpkg.MailClient
	database     *ormpkg.PostgresClient
	config       *Config
}

func NewWorker(logger *zap.Logger, brokerClient *eventpkg.KafkaClient, mailClient *clientpkg.MailClient, database *ormpkg.PostgresClient, config *Config) *Worker {
	context, cancel := context.WithCancel(context.Background())
	this := &Worker{
		context:      context,
		cancel:       cancel,
		logger:       logger,
		brokerClient: brokerClient,
		mailClient:   mailClient,
		database:     database,
		config:       config,
	}
	this.router = NewRouter(
		map[string][]EventHandler{
			eventpkg.AUTHORIZATION_LOGIN: {
				this.AuthorizationLoginHandler,
			},
			eventpkg.AUTHORIZATION_REQUEST_PASSWORD_RESET: {
				this.AuthorizationRequestPasswordResetHandler,
			},
			eventpkg.AUTHORIZATION_REGISTER: {
				this.AuthorizationRegisterHandler,
			},
		},
	)
	return this
}

func (this *Worker) Start() error {
	this.logger.Info("starting mail worker")

	this.waitGroup.Add(1)
	go this.worker()
	return nil
}

func (this *Worker) Stop() error {
	this.logger.Info("stopping mail worker")

	this.cancel()
	this.waitGroup.Wait()
	return nil
}

func (this *Worker) worker() {
	defer this.waitGroup.Done()

	for {
		select {
		case <-this.context.Done():
			return
		case <-time.After(1 * time.Millisecond):
		}

		event, data, err := this.brokerClient.ReadMessage(this.context)
		if err != nil {
			this.logger.Error("error receiving kafka message", zap.Error(err))
			continue
		}

		err = this.router.Handle(event, []byte(data))
		if err != nil {
			this.logger.Error("error handling kafka message", zap.Error(err))
			continue
		}
	}
}

func (this *Worker) AuthorizationRegisterHandler(data []byte) error {
	var message eventpkg.AuthorizationRegisterMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(message.ID)
	if err != nil {
		return err
	}

	user, err := this.database.SelectUserByID(userID.String())
	if err != nil {
		return err
	}

	verificationURL := fmt.Sprintf("%s?token=%s", this.config.VerificationURL, user.VerificationToken)

	fromEmail := "no-reply@stormhead.org" // Placeholder, should be configurable
	subject := "Email Verification"

	templateData := struct {
		User string
		URL  string
		Time string
	}{
		User: user.Name,
		URL:  verificationURL,
		Time: "24", // Placeholder for expiration time
	}

	content, err := templatepkg.Render("template/mail_confirm.html", templateData)
	if err != nil {
		return err
	}

	err = this.mailClient.SendHTML(fromEmail, user.Email, subject, content)
	if err != nil {
		return err
	}

	this.logger.Info("sent email verification email", zap.String("email", user.Email))
	return nil
}

func (this *Worker) AuthorizationLoginHandler(data []byte) error {
	var message eventpkg.AuthorizationLoginMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return err
	}

	this.logger.Info("user logged in", zap.String("id", message.ID))
	return nil
}

func (this *Worker) AuthorizationRequestPasswordResetHandler(data []byte) error {
	var message eventpkg.AuthorizationRequestPasswordReset
	err := json.Unmarshal(data, &message)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(message.ID)
	if err != nil {
		return err
	}

	user, err := this.database.SelectUserByID(userID.String())
	if err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s?token=%s", this.config.PasswordResetURL, user.ResetToken)

	fromEmail := "no-reply@stormhead.org" // Placeholder, should be configurable
	subject := "Password Reset Request"

	templateData := struct {
		User string
		URL  string
		Time string
	}{
		User: user.Name,
		URL:  resetURL,
		Time: "1", // Placeholder for expiration time
	}

	content, err := templatepkg.Render("template/mail_recover.html", templateData)
	if err != nil {
		return err
	}

	err = this.mailClient.SendHTML(fromEmail, user.Email, subject, content)
	if err != nil {
		return err
	}

	this.logger.Info("sent password reset email", zap.String("email", user.Email))
	return nil
}
