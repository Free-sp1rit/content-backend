package handler

type RegisterResponse struct {
	ID int64 `json:"id"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
