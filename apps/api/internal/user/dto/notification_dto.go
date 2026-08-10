package dto

type NotificationPreferenceResponse struct {
	MutedTypes []string `json:"muted_types"`
}

type UpdateNotificationPreferenceRequest struct {
	MutedTypes []string `json:"muted_types"`
}
