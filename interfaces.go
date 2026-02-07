//go:build !tinygo

package api

type SenderService interface {
	GetSession() *Session
	GetService() *Service
}

