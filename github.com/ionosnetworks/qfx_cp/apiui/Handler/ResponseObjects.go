package Handler

type UserInfo struct {
	Userrole  string `json:"userrole"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Result    string `json:"result"`
	Sessionid string `json:"sessionid"`
	Error     string `json:"error"`
}
