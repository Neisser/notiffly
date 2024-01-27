package notification

import "notifly/pkg/user"

type userModel = user.Model

type Model struct {
	From    userModel `json:"from"`
	To      userModel `json:"to"`
	Message string    `json:"message"`
}
