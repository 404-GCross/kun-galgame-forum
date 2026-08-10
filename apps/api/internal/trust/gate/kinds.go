package gate

import "kun-galgame-api/pkg/trustclient"

const (
	SubjectKindGalgameComment = "galgame_comment"
	SubjectKindGalgame        = "galgame"
	SubjectKindUser           = "user"
)

var CanonicalSubjectKinds = []string{
	SubjectKindTopic,
	SubjectKindReply,
	SubjectKindTopicComment,
	SubjectKindTopicPoll,
	SubjectKindGalgameRating,
	SubjectKindGalgameResource,
	SubjectKindGalgameCollection,
	SubjectKindGalgameQuiz,
	SubjectKindGalgameQuizAnswer,
	SubjectKindToolset,
	SubjectKindToolsetResource,
	SubjectKindGalgameComment,
	SubjectKindGalgame,
	SubjectKindUser,
}

func CanonicalSubjectKindItems() []trustclient.EnsureSubjectKindItem {
	items := make([]trustclient.EnsureSubjectKindItem, len(CanonicalSubjectKinds))
	for i, k := range CanonicalSubjectKinds {
		items[i] = trustclient.EnsureSubjectKindItem{Key: k}
	}
	return items
}
