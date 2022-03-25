package service

import (
	"github.com/DagDigg/unpaper/backend/pkg/chat"
	"github.com/DagDigg/unpaper/backend/pkg/chat/controller"
	"github.com/DagDigg/unpaper/backend/pkg/chat/usecase"
	"github.com/go-redis/redis/v8"
)

type Chat struct {
	controller chat.Controller
}

func New(rdb *redis.Client) chat.Controller {
	ucs := usecase.New(rdb)
	ctrl := controller.New(ucs)

	return ctrl
}
