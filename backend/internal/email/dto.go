package email

// VerifyEmailRequest is the payload for email verification request.
type VerifyEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

// VerifyEmailResponse is the response for email verification.
type VerifyEmailResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

// ResetPasswordRequest is the payload for password reset request.
type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
}

// ResetPasswordResponse is the response for password reset.
type ResetPasswordResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

// WelcomeRequest is the payload for welcome email.
type WelcomeRequest struct {
	Email      string `json:"email" binding:"required,email"`
	FullName   string `json:"full_name" binding:"required"`
	LoginURL   string `json:"login_url" binding:"required"`
}

// WelcomeResponse is the response for welcome email.
type WelcomeResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

// ShareMemoryRequest is the payload for share memory email.
type ShareMemoryRequest struct {
	MemoryID string `json:"memory_id" binding:"required,uuid"`
	ViewerName string `json:"viewer_name" binding:"required"`
	MemoryTitle string `json:"memory_title" binding:"required"`
}

// ShareMemoryResponse is the response for share memory email.
type ShareMemoryResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

// MemoryViewedRequest is the payload for memory viewed email.
type MemoryViewedRequest struct {
	MemoryID  string `json:"memory_id" binding:"required,uuid"`
	ViewerName string `json:"viewer_name" binding:"required"`
	MemoryTitle string `json:"memory_title" binding:"required"`
}

// MemoryViewedResponse is the response for memory viewed email.
type MemoryViewedResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

// ReactionReceivedRequest is the payload for reaction received email.
type ReactionReceivedRequest struct {
	MemoryID string `json:"memory_id" binding:"required,uuid"`
	ReactingUser string `json:"reacting_user" binding:"required"`
	ReactionType string `json:"reaction_type" binding:"required"`
	TargetUser string `json:"target_user" binding:"required"`
}

// ReactionReceivedResponse is the response for reaction received email.
type ReactionReceivedResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
}

// StorageWarningRequest is the payload for storage warning email.
type StorageWarningRequest struct {
	UserStorageUsed int64 `json:"user_storage_used" binding:"required"`
	UserStorageQuota int64 `json:"user_storage_quota" binding:"required"`
	UsagePercent int `json:"usage_percent" binding:"required"`
}

// StorageWarningResponse is the response for storage warning email.
type StorageWarningResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	DeleteOption bool `json:"delete_option"`
}