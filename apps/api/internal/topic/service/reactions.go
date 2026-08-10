package service

import (
	"kun-galgame-api/internal/topic/dto"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/userclient"
)

func reactionReactorIDs(rows []repository.ReactionRow) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.UserID > 0 {
			ids = append(ids, row.UserID)
		}
	}
	return ids
}

func buildReactionSummaries(
	rows []repository.ReactionRow,
	mine []string,
	userMap map[int]userclient.User,
) []dto.ReactionSummary {
	mineSet := make(map[string]bool, len(mine))
	for _, k := range mine {
		mineSet[k] = true
	}
	idx := map[string]int{}
	out := []dto.ReactionSummary{}
	for _, row := range rows {
		i, ok := idx[row.Reaction]
		if !ok {
			i = len(out)
			idx[row.Reaction] = i
			out = append(out, dto.ReactionSummary{
				Reaction: row.Reaction,
				Count:    row.Cnt,
				Mine:     mineSet[row.Reaction],
			})
		}
		if u, ok := userMap[row.UserID]; ok {
			out[i].Reactors = append(out[i].Reactors,
				dto.KunUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar})
		}
	}
	return out
}

func replyReactorIDs(rows []repository.ReplyReactionRow) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.UserID > 0 {
			ids = append(ids, row.UserID)
		}
	}
	return ids
}

func buildRepliesReactions(
	rows []repository.ReplyReactionRow,
	mineByReply map[int][]string,
	userMap map[int]userclient.User,
) map[int][]dto.ReactionSummary {
	byReply := map[int][]repository.ReactionRow{}
	for _, row := range rows {
		byReply[row.TopicReplyID] = append(byReply[row.TopicReplyID],
			repository.ReactionRow{Reaction: row.Reaction, UserID: row.UserID, Cnt: row.Cnt})
	}
	out := make(map[int][]dto.ReactionSummary, len(byReply))
	for replyID, rrows := range byReply {
		out[replyID] = buildReactionSummaries(rrows, mineByReply[replyID], userMap)
	}
	return out
}
