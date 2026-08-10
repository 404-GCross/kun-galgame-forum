package dto

type UploadInitRequest struct {
	Filename    string `json:"filename" validate:"required"`
	FileSize    int64  `json:"filesize" validate:"required,min=1"`
	ContentType string `json:"content_type"`
}

type UploadCompletePart struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
}

type UploadCompleteRequest struct {
	ArtifactUUID string               `json:"artifact_uuid" validate:"required"`
	Parts        []UploadCompletePart `json:"parts"`
}

type UploadAbortRequest struct {
	ArtifactUUID string `json:"artifact_uuid" validate:"required"`
}

type UploadResumeRequest struct {
	ArtifactUUID string `json:"artifact_uuid" validate:"required"`
}

type UploadInitPart struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
}

type UploadInitResponse struct {
	ArtifactUUID string           `json:"artifact_uuid"`
	Multipart    bool             `json:"multipart"`
	UploadURL    string           `json:"upload_url,omitempty"`
	PartSize     int64            `json:"part_size,omitempty"`
	Parts        []UploadInitPart `json:"parts,omitempty"`
	ExpiresAt    string           `json:"expires_at"`
}

type UploadCompleteResponse struct {
	ArtifactUUID string `json:"artifact_uuid"`
	Size         int64  `json:"size"`
}

type UploadResumePart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size"`
}

type UploadResumeResponse struct {
	ArtifactUUID  string             `json:"artifact_uuid"`
	Multipart     bool               `json:"multipart"`
	UploadURL     string             `json:"upload_url,omitempty"`
	PartSize      int64              `json:"part_size,omitempty"`
	Parts         []UploadInitPart   `json:"parts,omitempty"`
	UploadedParts []UploadResumePart `json:"uploaded_parts,omitempty"`
	ExpiresAt     string             `json:"expires_at"`
}
