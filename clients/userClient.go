package clients

import (
	"github.com/sirupsen/logrus"
	"os/user"
)

type UserClient interface {
	initialize(logger *logrus.Logger)
	GetUserByName(username string) (user.User, error)
}

type UserClientImpl struct {
	logger *logrus.Logger
}

func (userClient *UserClientImpl) initialize(logger *logrus.Logger) {
	userClient.logger = logger
}

func (userClient *UserClientImpl) GetUserByName(username string) (user.User, error) {
	return user.Lookup(username)
}
