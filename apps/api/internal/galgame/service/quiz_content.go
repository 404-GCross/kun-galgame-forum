package service

import (
	"encoding/json"
	"strings"

	"kun-galgame-api/pkg/errors"
)

// This file owns the quiz `content` JSONB payload: its per-type shape, the
// server-side validation on create, the answer-key stripping for read paths,
// and the grading. The correct-answer key lives only in the stored `content`
// and is NEVER sent to a client that has not answered — grading happens here,
// on the server.

// Stored `content` shapes (authored payload, incl. the answer key).
type (
	quizSingleContent struct {
		Options []string `json:"options"`
		Answer  int      `json:"answer"`
	}
	quizMultipleContent struct {
		Options []string `json:"options"`
		Answers []int    `json:"answers"`
	}
	quizJudgeContent struct {
		Answer bool `json:"answer"`
	}
	quizFillBlank struct {
		Accepted []string `json:"accepted"`
	}
	quizFillContent struct {
		Blanks []quizFillBlank `json:"blanks"`
	}
	quizEssayContent struct {
		Reference string `json:"reference"`
	}
)

// Submitted answer shapes (from the client).
type (
	quizSingleSubmit struct {
		Value int `json:"value"`
	}
	quizMultipleSubmit struct {
		Values []int `json:"values"`
	}
	quizJudgeSubmit struct {
		Value bool `json:"value"`
	}
	quizFillSubmit struct {
		Values []string `json:"values"`
	}
)

// Quiz type constants.
const (
	quizTypeSingle   = "single"
	quizTypeMultiple = "multiple"
	quizTypeJudge    = "judge"
	quizTypeFill     = "fill"
	quizTypeEssay    = "essay"
)

// validateQuizContent shape-checks an authored payload against its type. Keeps
// garbage / unanswerable questions out of the table.
func validateQuizContent(qtype string, raw json.RawMessage) *errors.AppError {
	switch qtype {
	case quizTypeSingle:
		var c quizSingleContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return errors.ErrBadRequest("题目内容格式错误")
		}
		if len(c.Options) < 2 {
			return errors.ErrBadRequest("单选题至少需要 2 个选项")
		}
		if c.Answer < 0 || c.Answer >= len(c.Options) {
			return errors.ErrBadRequest("正确答案超出选项范围")
		}
	case quizTypeMultiple:
		var c quizMultipleContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return errors.ErrBadRequest("题目内容格式错误")
		}
		if len(c.Options) < 2 {
			return errors.ErrBadRequest("多选题至少需要 2 个选项")
		}
		if len(c.Answers) == 0 {
			return errors.ErrBadRequest("多选题至少需要 1 个正确答案")
		}
		for _, a := range c.Answers {
			if a < 0 || a >= len(c.Options) {
				return errors.ErrBadRequest("正确答案超出选项范围")
			}
		}
	case quizTypeJudge:
		var c quizJudgeContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return errors.ErrBadRequest("题目内容格式错误")
		}
	case quizTypeFill:
		var c quizFillContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return errors.ErrBadRequest("题目内容格式错误")
		}
		if len(c.Blanks) == 0 {
			return errors.ErrBadRequest("填空题至少需要 1 个空")
		}
		for _, b := range c.Blanks {
			if !hasNonEmpty(b.Accepted) {
				return errors.ErrBadRequest("每个空至少需要 1 个可接受答案")
			}
		}
	case quizTypeEssay:
		var c quizEssayContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return errors.ErrBadRequest("题目内容格式错误")
		}
		if strings.TrimSpace(c.Reference) == "" {
			return errors.ErrBadRequest("问答题需要提供参考答案")
		}
	default:
		return errors.ErrBadRequest("未知的题目类型")
	}
	return nil
}

// stripQuizContent returns the PUBLIC payload safe to send before answering —
// options are kept (they are the question), the answer key is dropped.
func stripQuizContent(qtype string, raw json.RawMessage) json.RawMessage {
	switch qtype {
	case quizTypeSingle:
		var c quizSingleContent
		_ = json.Unmarshal(raw, &c)
		return mustJSON(map[string]any{"options": c.Options})
	case quizTypeMultiple:
		var c quizMultipleContent
		_ = json.Unmarshal(raw, &c)
		return mustJSON(map[string]any{"options": c.Options})
	case quizTypeFill:
		var c quizFillContent
		_ = json.Unmarshal(raw, &c)
		return mustJSON(map[string]any{"blank_count": len(c.Blanks)})
	default: // judge / essay carry no pre-answer payload
		return json.RawMessage("{}")
	}
}

// gradeQuiz grades a submitted answer against the stored content. Returns
// (nil, nil) for essay (ungraded). A malformed submission is a 400.
func gradeQuiz(qtype string, content, submitted json.RawMessage) (*bool, *errors.AppError) {
	switch qtype {
	case quizTypeSingle:
		var c quizSingleContent
		var s quizSingleSubmit
		if json.Unmarshal(content, &c) != nil || json.Unmarshal(submitted, &s) != nil {
			return nil, errors.ErrBadRequest("答案格式错误")
		}
		return boolPtr(s.Value == c.Answer), nil
	case quizTypeMultiple:
		var c quizMultipleContent
		var s quizMultipleSubmit
		if json.Unmarshal(content, &c) != nil || json.Unmarshal(submitted, &s) != nil {
			return nil, errors.ErrBadRequest("答案格式错误")
		}
		return boolPtr(intSetEqual(c.Answers, s.Values)), nil
	case quizTypeJudge:
		var c quizJudgeContent
		var s quizJudgeSubmit
		if json.Unmarshal(content, &c) != nil || json.Unmarshal(submitted, &s) != nil {
			return nil, errors.ErrBadRequest("答案格式错误")
		}
		return boolPtr(s.Value == c.Answer), nil
	case quizTypeFill:
		var c quizFillContent
		var s quizFillSubmit
		if json.Unmarshal(content, &c) != nil || json.Unmarshal(submitted, &s) != nil {
			return nil, errors.ErrBadRequest("答案格式错误")
		}
		if len(s.Values) != len(c.Blanks) {
			return boolPtr(false), nil
		}
		for i, blank := range c.Blanks {
			if !fillMatches(blank.Accepted, s.Values[i]) {
				return boolPtr(false), nil
			}
		}
		return boolPtr(true), nil
	default: // essay — not graded
		return nil, nil
	}
}

// ──────────────────────────────────────────
// helpers
// ──────────────────────────────────────────

// normalizeFillAnswer canonicalizes a fill-in answer for comparison: trims,
// lowercases, and drops all whitespace so "Fate stay night" == "fatestaynight".
func normalizeFillAnswer(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), "")
}

func fillMatches(accepted []string, got string) bool {
	g := normalizeFillAnswer(got)
	if g == "" {
		return false
	}
	for _, a := range accepted {
		if normalizeFillAnswer(a) == g {
			return true
		}
	}
	return false
}

func hasNonEmpty(ss []string) bool {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}

// intSetEqual reports whether two int slices hold the same set (order- and
// duplicate-insensitive).
func intSetEqual(a, b []int) bool {
	set := make(map[int]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	seen := make(map[int]struct{}, len(b))
	for _, v := range b {
		if _, ok := set[v]; !ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return len(seen) == len(set)
}

func boolPtr(b bool) *bool { return &b }

func mustJSON(v any) json.RawMessage {
	out, _ := json.Marshal(v)
	return out
}
