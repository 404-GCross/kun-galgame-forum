import type { ForumPermission } from '~/composables/useCan'

export type CommunityCommentTarget =
  | { kind: 'galgame'; galgameId: number }
  | { kind: 'rating'; ratingId: number }
  | { kind: 'website'; websiteId: number; domain: string }
  | { kind: 'toolset'; toolsetId: number }
  | { kind: 'resource'; resourceId: number }
  | { kind: 'quiz'; quizId: number }

export interface CommunityCommentSurface {
  key: CommunityCommentTarget['kind']
  maxLength: number
  anchorPrefix: string
  deletePermission: ForumPermission
  composerPlaceholder: string
  isFlat: boolean
  showsReplyTarget: boolean
  supportsMentions: boolean
  listUrl: string
  addressQuery: Record<string, number | string>
  editUrl: (postId: number) => string
  editQuery: Record<string, number | string>
  deleteUrl: (postId: number) => string
  deleteQuery: Record<string, number | string>
  createBody: (
    content: string,
    replyToPostId: number | null,
    targetUserId?: number
  ) => Record<string, unknown>
}

const MENTION_PLACEHOLDER =
  '请温柔的发表你的看法吧～「评论给」已废除，@用户名 即可通知对方'

export const communityCommentSurface = (
  target: CommunityCommentTarget
): CommunityCommentSurface => {
  switch (target.kind) {
    case 'galgame': {
      const base = `/galgame/${target.galgameId}/comments`
      return {
        key: 'galgame',
        maxLength: 5000,
        anchorPrefix: 'galgame-comment',
        deletePermission: 'comment.galgame.delete',
        composerPlaceholder: MENTION_PLACEHOLDER,
        isFlat: false,
        showsReplyTarget: false,
        supportsMentions: true,
        listUrl: base,
        addressQuery: {},
        editUrl: (postId) => `/galgame/comments/${postId}`,
        editQuery: { gid: target.galgameId },
        deleteUrl: (postId) => `/galgame/comments/${postId}`,
        deleteQuery: { gid: target.galgameId },
        createBody: (content, replyToPostId) => ({
          content,
          reply_to_post_id: replyToPostId
        })
      }
    }

    case 'rating':
      return {
        key: 'rating',
        maxLength: 1314,
        anchorPrefix: 'rating-comment',
        deletePermission: 'comment.rating.delete',
        composerPlaceholder: '发布对这个评分的观点，请不要锐评',
        isFlat: true,
        showsReplyTarget: true,
        supportsMentions: false,
        listUrl: `/galgame-rating/${target.ratingId}/comments`,
        addressQuery: {},
        editUrl: (postId) => `/galgame/comments/${postId}`,
        editQuery: {},
        deleteUrl: (postId) =>
          `/galgame-rating/${target.ratingId}/comments/${postId}`,
        deleteQuery: {},
        createBody: (content, _replyToPostId, targetUserId) => ({
          content,
          target_user_id: targetUserId
        })
      }

    case 'website':
      return {
        key: 'website',
        maxLength: 1007,
        anchorPrefix: 'website-comment',
        deletePermission: 'comment.website.delete',
        composerPlaceholder: '说说你对这个网站的看法吧～',
        isFlat: false,
        showsReplyTarget: true,
        supportsMentions: false,
        listUrl: `/website/${target.domain}/comments`,
        addressQuery: { website_id: target.websiteId },
        editUrl: (postId) => `/galgame/comments/${postId}`,
        editQuery: {},
        deleteUrl: (postId) => `/website/${target.domain}/comments/${postId}`,
        deleteQuery: { website_id: target.websiteId },
        createBody: (content, replyToPostId) => ({
          content,
          website_id: target.websiteId,
          reply_to_post_id: replyToPostId
        })
      }

    case 'toolset':
      return {
        key: 'toolset',
        maxLength: 1007,
        anchorPrefix: 'toolset-comment',
        deletePermission: 'comment.toolset.delete',
        composerPlaceholder: '对这个工具有任何使用疑问，都可以在这里提出～',
        isFlat: false,
        showsReplyTarget: true,
        supportsMentions: false,
        listUrl: `/toolset/${target.toolsetId}/comments`,
        addressQuery: {},
        editUrl: (postId) => `/galgame/comments/${postId}`,
        editQuery: {},
        deleteUrl: (postId) =>
          `/toolset/${target.toolsetId}/comments/${postId}`,
        deleteQuery: {},
        createBody: (content, replyToPostId) => ({
          content,
          reply_to_post_id: replyToPostId
        })
      }

    case 'resource':
      return {
        key: 'resource',
        maxLength: 1007,
        anchorPrefix: 'resource-comment',
        deletePermission: 'comment.resource.delete',
        composerPlaceholder: '这个资源能正常使用吗？有问题可以在这里反馈～',
        isFlat: false,
        showsReplyTarget: true,
        supportsMentions: false,
        listUrl: `/galgame-resource/${target.resourceId}/comments`,
        addressQuery: {},
        editUrl: (postId) => `/galgame/comments/${postId}`,
        editQuery: {},
        deleteUrl: (postId) =>
          `/galgame-resource/${target.resourceId}/comments/${postId}`,
        deleteQuery: {},
        createBody: (content, replyToPostId) => ({
          content,
          reply_to_post_id: replyToPostId
        })
      }

    case 'quiz':
      return {
        key: 'quiz',
        maxLength: 1007,
        anchorPrefix: 'quiz-comment',
        deletePermission: 'comment.quiz.delete',
        composerPlaceholder: '聊聊这道题目吧～请不要直接剧透答案',
        isFlat: false,
        showsReplyTarget: true,
        supportsMentions: false,
        listUrl: `/galgame-quiz/${target.quizId}/comments`,
        addressQuery: {},
        editUrl: (postId) => `/galgame/comments/${postId}`,
        editQuery: {},
        deleteUrl: (postId) =>
          `/galgame-quiz/${target.quizId}/comments/${postId}`,
        deleteQuery: {},
        createBody: (content, replyToPostId) => ({
          content,
          reply_to_post_id: replyToPostId
        })
      }
  }
}
