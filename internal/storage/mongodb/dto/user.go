package dto

type User struct {
	ID           string `bson:"_id,omitempty"`
	FullName     string `bson:"full_name"`
	Username     string `bson:"username"`
	PasswordHash string `bson:"password_hash"`
}

type CreateUserReq struct {
	FullName string
	Username string
	Password string
}

type GetUserByUsernameReq struct {
	Username string
}

type GetUserByUsernameResp struct {
	ID           string
	FullName     string
	Username     string
	PasswordHash string
}

type GetUsersReq struct {
	ID     string
	Name   string
	Limit  int64
	Offset int64
}

type GetUsersResp struct {
	Users []User
}

type GetUserByIDReq struct {
	ID string
}

type GetUserByIDResp struct {
	ID       string
	FullName string
	Username string
}
