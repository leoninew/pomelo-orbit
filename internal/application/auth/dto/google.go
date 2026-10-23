package dto

type GoogleAuthorization struct {
	URL     string
	Session string
}

type GoogleCallbackInput struct {
	Code      string
	State     string
	Session   string
	IP        string
	UserAgent string
}

type GoogleLoginResult struct {
	Token    string
	Redirect string
}
