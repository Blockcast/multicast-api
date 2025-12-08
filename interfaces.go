package api

type SenderService interface {
	GetSession() *Session
	GetService() *Service
}

