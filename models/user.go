package models

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Key_fp   string `json:"key_fp"`
	Dvc_mac  string `json:"dvc_mac"`
}
