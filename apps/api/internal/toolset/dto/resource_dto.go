package dto

import (
	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
)

type ResourceDetailRequest struct {
	ResourceID int `query:"toolset_resource_id" validate:"required,min=1"`
}

type CreateResourceRequest struct {
	Content      string `json:"content" validate:"max=1007"`
	Type         string `json:"type" validate:"required,oneof=s3 user"`
	ArtifactUUID string `json:"artifact_uuid" validate:"max=36"`
	Code         string `json:"code" validate:"max=1007"`
	Password     string `json:"password" validate:"max=1007"`
	Size         string `json:"size" validate:"max=107"`
	Note         string `json:"note" validate:"max=1007"`
}

type UpdateResourceRequest struct {
	ResourceID int    `json:"toolset_resource_id" validate:"required,min=1"`
	Content    string `json:"content" validate:"max=1007"`
	Code       string `json:"code" validate:"max=1007"`
	Password   string `json:"password" validate:"max=1007"`
	Size       string `json:"size" validate:"max=107"`
	Note       string `json:"note" validate:"max=1007"`
}

type DeleteResourceRequest struct {
	ResourceID int `query:"toolset_resource_id" validate:"required,min=1"`
}

type ResourceDetailResponse struct {
	model.GalgameToolsetResource
	User userModel.UserBrief `json:"user"`
}

type CreatedResourceResponse = model.GalgameToolsetResource
