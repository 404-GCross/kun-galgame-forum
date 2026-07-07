package service

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type QuizService struct {
	quizRepo   *repository.QuizRepository
	wikiClient *client.GalgameClient
	userClient *userclient.Client
	helpers    InteractionHelpers
}

func NewQuizService(
	quizRepo *repository.QuizRepository,
	wikiClient *client.GalgameClient,
	userClient *userclient.Client,
) *QuizService {
	return &QuizService{
		quizRepo:   quizRepo,
		wikiClient: wikiClient,
		userClient: userClient,
	}
}

// quizCreateReward: flat authoring reward (被采纳), granted at create time and
// refunded on delete. Difficulty is ignored for now (kept in the signature so
// re-introducing a scaled reward stays a one-line change).
func quizCreateReward(_ int) int {
	return constants.QuizCreateReward
}

// quizCorrectReward: reward for a correct answer. Currently 0 (disabled).
func quizCorrectReward(_ int) int {
	return constants.QuizCorrectReward
}

// ──────────────────────────────────────────
// GetAllQuizzes — GET /galgame-quiz/all
// ──────────────────────────────────────────

func (s *QuizService) GetAllQuizzes(
	ctx context.Context,
	req *dto.QuizListRequest,
	isSFW bool,
) (*dto.QuizListPage, *errors.AppError) {
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	rows, total := s.quizRepo.ListPaginated(model.QuizFilter{
		Category:   req.Category,
		Type:       req.Type,
		SortField:  req.SortField,
		SortOrder:  sortOrder,
		Difficulty: req.Difficulty,
		GalgameID:  req.GalgameID,
		UserID:     req.UserID,
		Page:       req.Page,
		Limit:      req.Limit,
	})
	return s.hydrateCards(ctx, rows, total, isSFW), nil
}

// GetMyAnswered — GET /galgame-quiz/mine/answered (self only).
func (s *QuizService) GetMyAnswered(
	ctx context.Context,
	userID, page, limit int,
) (*dto.QuizListPage, *errors.AppError) {
	rows, total := s.quizRepo.ListAnsweredByUser(userID, page, limit)
	// A self list is not SFW-gated (the viewer is logged in and answered these).
	return s.hydrateCards(ctx, rows, total, false), nil
}

// hydrateCards resolves authors + linked-game briefs and drops rows whose
// author is banned or whose linked game is SFW-filtered. General-trivia rows
// (galgame_id NULL) are always kept.
func (s *QuizService) hydrateCards(
	ctx context.Context,
	rows []model.GalgameQuizRow,
	total int64,
	isSFW bool,
) *dto.QuizListPage {
	userIDs := make([]int, 0, len(rows))
	galgameIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.UserID)
		if r.GalgameID != nil {
			galgameIDs = append(galgameIDs, *r.GalgameID)
		}
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	briefMap := s.fetchBriefsPublic(ctx, galgameIDs, isSFW)

	cards := make([]dto.QuizCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		var brief *client.GalgameBrief
		if r.GalgameID != nil {
			b, ok := briefMap[*r.GalgameID]
			if !ok {
				continue // linked game SFW-filtered → drop
			}
			brief = &b
		}
		cards = append(cards, quizRowToCard(r, u, brief))
	}
	return &dto.QuizListPage{QuizData: cards, Total: total}
}

// ──────────────────────────────────────────
// GetQuizPlay — GET /galgame-quiz/:id
// ──────────────────────────────────────────

func (s *QuizService) GetQuizPlay(
	ctx context.Context,
	quizID, currentUserID int,
) (*dto.QuizPlay, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(quizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	// Hide a banned author's quiz even by direct link.
	author, _, _ := s.userClient.User(ctx, quiz.UserID)
	if !userclient.IsRenderable(author) {
		return nil, errors.ErrNotFound("题目不存在")
	}

	s.quizRepo.IncrementView(quizID)
	quiz.View++

	isAuthor := currentUserID != 0 && currentUserID == quiz.UserID

	// Reveal the answer key + explanation only once the viewer has a row
	// (answered, or the author's auto-row).
	var myAnswer *dto.QuizAnswerResult
	if currentUserID != 0 {
		if row, has := s.quizRepo.FindAnswer(quizID, currentUserID); has {
			myAnswer = &dto.QuizAnswerResult{
				Submitted:     row.Submitted,
				IsCorrect:     row.IsCorrect,
				Answer:        quiz.Content,
				Explanation:   quiz.Explanation,
				QualityRating: row.QualityRating,
			}
		}
	}

	play := &dto.QuizPlay{
		ID:           quiz.ID,
		User:         userBriefToDTO(author),
		Category:     quiz.Category,
		SpoilerLevel: quiz.SpoilerLevel,
		Type:         quiz.Type,
		Difficulty:   quiz.Difficulty,
		Question:     quiz.Question,
		Content:      stripQuizContent(quiz.Type, quiz.Content),
		QuizStats:    quizStats(quiz.View, quiz.AnswerCount, quiz.CorrectCount, quiz.QualitySum, quiz.QualityCount),
		Created:      quiz.CreatedAt.Format(time.RFC3339),
		Updated:      quiz.UpdatedAt.Format(time.RFC3339),
		Galgame:      s.briefFor(ctx, quiz.GalgameID),
		IsAuthor:     isAuthor,
		MyAnswer:     myAnswer,
	}
	return play, nil
}

// ──────────────────────────────────────────
// CreateQuiz — POST /galgame-quiz
// Anyone publishes immediately (no review gate in MVP). Grants the author the
// difficulty-scaled authoring reward, and seeds the author's roster row.
// ──────────────────────────────────────────

func (s *QuizService) CreateQuiz(
	ctx context.Context,
	userID int,
	req *dto.CreateQuizRequest,
) (*dto.CreatedQuiz, *errors.AppError) {
	if appErr := validateQuizContent(req.Type, req.Content); appErr != nil {
		return nil, appErr
	}

	spoiler := req.SpoilerLevel
	if spoiler == "" {
		spoiler = "none"
	}
	quiz := &model.GalgameQuiz{
		UserID:       userID,
		GalgameID:    req.GalgameID,
		Category:     req.Category,
		SpoilerLevel: spoiler,
		Type:         req.Type,
		Difficulty:   req.Difficulty,
		Question:     req.Question,
		Content:      req.Content,
		Explanation:  req.Explanation,
	}
	reward := quizCreateReward(req.Difficulty)

	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.Create(tx, quiz); err != nil {
			return err
		}
		// Author roster row: marks them a participant so they can't answer
		// their own question; carries no submitted answer / grade.
		if err := s.quizRepo.CreateAnswer(tx, &model.GalgameQuizAnswer{
			QuizID: quiz.ID, UserID: userID, Role: "author",
		}); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, reward,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_quiz", quiz.ID))
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建题目失败")
	}

	author, _, _ := s.userClient.User(ctx, userID)
	return &dto.CreatedQuiz{
		ID:           quiz.ID,
		User:         userBriefToDTO(author),
		Category:     quiz.Category,
		SpoilerLevel: quiz.SpoilerLevel,
		Type:         quiz.Type,
		Difficulty:   quiz.Difficulty,
		Question:     quiz.Question,
		QuizStats:    quizStats(0, 0, 0, 0, 0),
		Created:      quiz.CreatedAt.Format(time.RFC3339),
		Updated:      quiz.UpdatedAt.Format(time.RFC3339),
		Galgame:      s.briefFor(ctx, quiz.GalgameID),
	}, nil
}

// ──────────────────────────────────────────
// AnswerQuiz — POST /galgame-quiz/:id/answer
// One attempt per user. Grades server-side; a correct answer grants the
// difficulty-scaled reward once. Reveals the answer key + explanation.
// ──────────────────────────────────────────

func (s *QuizService) AnswerQuiz(
	userID int,
	req *dto.AnswerQuizRequest,
) (*dto.QuizAnswerResult, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(req.QuizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	if existing, has := s.quizRepo.FindAnswer(req.QuizID, userID); has {
		if existing.Role == "author" {
			return nil, errors.ErrBadRequest("不能回答自己出的题目")
		}
		return nil, errors.ErrBadRequest("您已经回答过该题目了")
	}

	grade, appErr := gradeQuiz(quiz.Type, quiz.Content, req.Submitted)
	if appErr != nil {
		return nil, appErr
	}
	correct := grade != nil && *grade
	reward := 0
	if correct {
		reward = quizCorrectReward(quiz.Difficulty)
	}

	row := &model.GalgameQuizAnswer{
		QuizID:    req.QuizID,
		UserID:    userID,
		Role:      "answerer",
		Submitted: req.Submitted,
		IsCorrect: grade,
		Rewarded:  reward > 0,
	}
	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.CreateAnswer(tx, row); err != nil {
			return err
		}
		if err := s.quizRepo.BumpAnswerStats(tx, req.QuizID, correct); err != nil {
			return err
		}
		if reward > 0 {
			s.helpers.AdjustMoemoepoint(tx, userID, reward,
				moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_quiz_answer", row.ID))
		}
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("提交答案失败")
	}

	return &dto.QuizAnswerResult{
		Submitted:   req.Submitted,
		IsCorrect:   grade,
		Answer:      quiz.Content,
		Explanation: quiz.Explanation,
		RewardDelta: reward,
	}, nil
}

// ──────────────────────────────────────────
// RateQuizQuality — PUT /galgame-quiz/:id/quality
// Only a genuine answerer may rate (not the author). Re-rating is allowed.
// ──────────────────────────────────────────

func (s *QuizService) RateQuizQuality(
	userID int,
	req *dto.RateQuizQualityRequest,
) (*dto.QuizQualityResult, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(req.QuizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	row, has := s.quizRepo.FindAnswer(req.QuizID, userID)
	if !has {
		return nil, errors.ErrForbidden("请先回答题目再评分")
	}
	if row.Role == "author" {
		return nil, errors.ErrForbidden("不能给自己出的题目评分")
	}

	sumDelta, countDelta := req.QualityRating, 1
	if row.QualityRating != nil {
		sumDelta, countDelta = req.QualityRating-*row.QualityRating, 0
	}

	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.SetAnswerQuality(tx, row.ID, req.QualityRating); err != nil {
			return err
		}
		return s.quizRepo.AdjustQuality(tx, req.QuizID, sumDelta, countDelta)
	})
	if txErr != nil {
		return nil, errors.ErrInternal("评分失败")
	}

	newSum := quiz.QualitySum + sumDelta
	newCount := quiz.QualityCount + countDelta
	return &dto.QuizQualityResult{
		QualityAverage: quizQualityAverage(newSum, newCount),
		QualityCount:   newCount,
		QualityRating:  req.QualityRating,
	}, nil
}

// ──────────────────────────────────────────
// DeleteQuiz — DELETE /galgame-quiz/:id
// Author or moderator. Refunds the author's create reward.
// ──────────────────────────────────────────

func (s *QuizService) DeleteQuiz(userID int, canModerate bool, quizID int) *errors.AppError {
	quiz, ok := s.quizRepo.FindByID(quizID)
	if !ok {
		return errors.ErrNotFound("题目不存在")
	}
	if quiz.UserID != userID && !canModerate {
		return errors.ErrForbidden("没有删除该题目的权限")
	}

	refund := -quizCreateReward(quiz.Difficulty)
	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.DeleteByID(tx, quizID); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, quiz.UserID, refund,
			moemoepoint.ReasonContentRemoved, moemoepoint.Ref("galgame_quiz", quizID))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除题目失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────

// briefFor returns the linked-game brief for detail paths (no SFW gate —
// mirrors the rating/galgame detail policy), or nil for general trivia.
func (s *QuizService) briefFor(ctx context.Context, galgameID *int) *dto.QuizGalgameBrief {
	if galgameID == nil {
		return nil
	}
	m := s.fetchBriefs(ctx, []int{*galgameID})
	b, ok := m[*galgameID]
	if !ok {
		return nil
	}
	return quizGalgameBrief(b)
}

func (s *QuizService) fetchBriefs(ctx context.Context, galgameIDs []int) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	m, _ := s.wikiClient.GetBatch(ctx, galgameIDs)
	if m == nil {
		return map[int]client.GalgameBrief{}
	}
	return m
}

func (s *QuizService) fetchBriefsPublic(ctx context.Context, galgameIDs []int, isSFW bool) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	m, _ := s.wikiClient.GetBatchPublic(ctx, galgameIDs, isSFW)
	if m == nil {
		return map[int]client.GalgameBrief{}
	}
	return m
}

// ──────────────────────────────────────────
// SearchGalgameOptions — GET /galgame-quiz/galgame-search
// Powers the 出题 modal's galgame picker: a name search enriched with each
// hit's banner + maker (会社) names. Soft-fails to an empty list.
// ──────────────────────────────────────────

// quizGalgameSearchLimit caps how many hits we enrich (one batch-detail call),
// keeping the picker snappy.
const quizGalgameSearchLimit = 12

type wikiGalgameSearchRow struct {
	ID int `json:"id"`
}

func (s *QuizService) SearchGalgameOptions(
	ctx context.Context, keywords string, isSFW bool,
) []dto.QuizGalgameOption {
	empty := []dto.QuizGalgameOption{}
	// Wiki's /series/search is the general galgame name search (returns any
	// matching galgame row) — the same one the FE series picker uses via
	// /galgame-series/search. We only need the ids here.
	data, appErr := s.wikiClient.Get(ctx, "/series/search", url.Values{"keywords": {keywords}})
	if appErr != nil {
		return empty
	}
	var rows []wikiGalgameSearchRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return empty
	}
	ids := make([]int, 0, quizGalgameSearchLimit)
	for _, r := range rows {
		if r.ID > 0 {
			ids = append(ids, r.ID)
		}
		if len(ids) >= quizGalgameSearchLimit {
			break
		}
	}
	if len(ids) == 0 {
		return empty
	}

	briefs := s.fetchDetailBriefs(ctx, ids, isSFW)
	options := make([]dto.QuizGalgameOption, 0, len(ids))
	for _, id := range ids {
		b, ok := briefs[id]
		if !ok {
			continue // SFW-filtered or missing
		}
		banner := b.EffectiveBannerURL
		if banner == "" {
			banner = b.Banner
		}
		officials := b.Officials
		if officials == nil {
			officials = []string{}
		}
		options = append(options, dto.QuizGalgameOption{
			ID: b.ID,
			Name: dto.KunLanguage{
				EnUs: b.NameEnUs, JaJp: b.NameJaJp,
				ZhCn: b.NameZhCn, ZhTw: b.NameZhTw,
			},
			Banner:          banner,
			BannerThumbhash: b.EffectiveBannerThumbhash,
			Officials:       officials,
		})
	}
	return options
}

func (s *QuizService) fetchDetailBriefs(ctx context.Context, ids []int, isSFW bool) map[int]client.GalgameDetailBrief {
	if len(ids) == 0 {
		return map[int]client.GalgameDetailBrief{}
	}
	m, _ := s.wikiClient.GetBatchDetailPublic(ctx, ids, isSFW)
	if m == nil {
		return map[int]client.GalgameDetailBrief{}
	}
	return m
}
