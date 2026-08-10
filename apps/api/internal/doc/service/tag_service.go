package service

import (
	"kun-galgame-api/internal/doc/dto"
	"kun-galgame-api/internal/doc/model"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/pkg/errors"
)

type TagService struct {
	tagRepo *repository.TagRepository
}

func NewTagService(tagRepo *repository.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

type TagListResult struct {
	Items []model.DocTag
	Total int64
}

func (s *TagService) GetList(req *dto.GetTagsRequest) *TagListResult {
	items, total := s.tagRepo.FindPaginated(req.Keyword, req.Page, req.Limit)
	return &TagListResult{Items: items, Total: total}
}

func (s *TagService) Create(req *dto.CreateTagRequest) (*model.DocTag, *errors.AppError) {
	tag := &model.DocTag{
		Slug:        req.Slug,
		Title:       req.Title,
		Description: req.Description,
	}
	if err := s.tagRepo.Create(tag); err != nil {
		return nil, errors.ErrInternal("创建标签失败")
	}
	return tag, nil
}

func (s *TagService) Delete(tagID int) *errors.AppError {
	if err := s.tagRepo.DeleteByID(tagID); err != nil {
		return errors.ErrInternal("删除标签失败")
	}
	return nil
}

func (s *TagService) Update(req *dto.UpdateTagRequest) (*model.DocTag, *errors.AppError) {
	tag, err := s.tagRepo.Update(req.TagID, map[string]any{
		"slug":        req.Slug,
		"title":       req.Title,
		"description": req.Description,
	})
	if err != nil {
		return nil, errors.ErrInternal("更新标签失败")
	}
	return tag, nil
}
