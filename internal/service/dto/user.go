package dto

type UserData struct {
	ID       string
	FullName string
	Username string
}

type RegisterReq struct {
	FullName string
	Username string
	Password string
}

type RegisterResp struct {
	ID string
}

type AuthenticateReq struct {
	Username string
	Password string
}

type AuthenticateResp struct {
	ID string
}
