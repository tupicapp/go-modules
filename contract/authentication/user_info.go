package authentication

// UserInfo holds the basic user information used by authentication services.
type UserInfo struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Username    string  `json:"username"`
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	CountryISO  string  `json:"country_iso"`
	Locale      string  `json:"locale"`
}
