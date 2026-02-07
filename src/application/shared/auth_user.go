package application_shared

type AuthUser struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
    Image  string `json:"image"`
}