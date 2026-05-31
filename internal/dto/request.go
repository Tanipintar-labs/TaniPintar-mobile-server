package dto

type RegisterRequest struct {
	Email                string `json:"email" binding:"required,email,max=255"`
	FullName             string `json:"full_name" binding:"required,min=2,max=255"`
	BirthPlace           string `json:"birth_place" binding:"required,min=2,max=255"`
	DateOfBirth          string `json:"date_of_birth" binding:"required"`
	Password             string `json:"password" binding:"required,min=8,max=72"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required,eqfield=Password"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
