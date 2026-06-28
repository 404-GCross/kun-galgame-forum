import type { GalgameCard } from './galgame'

export interface UserInfo {
  id: number
  name: string
  avatar: string
  // OAuth named-role set (empty for a plain user; `user` never appears). Drives
  // the profile badge via managementRoleLabel + a creator chip. `creator` is
  // orthogonal — no moderation/admin power.
  roles: string[]
  status: number
  moemoepoint: number
  bio: string
  created: Date | string

  upvote: number
  like: number
  dislike: number

  replyCreated: number
  commentCreated: number
  topic: number
  topicPoll: number

  galgame: number
  contributeGalgame: number
  galgameComment: number
  galgameRating: number

  galgameResource: number
  galgameToolset: number
  galgameToolsetResource: number

  dailyTopicCount: number
  dailyGalgameCount: number
}

export interface UserTopic {
  id: number
  title: string
  created: Date | string
}

export type UserGalgame = GalgameCard

export interface UserGalgameResource {
  id: number
  galgameId: number
  galgameName: KunLanguage
  type: string
  language: string
  platform: string
  size: string
  link: string[]
  code: string
  password: string
  note: string
  status: number
  created: Date | string
}

export interface UserReply {
  topicId: number
  // Floor → deep-link to this reply (/topic/:id?reply=<floor>).
  floor: number
  content: string
  created: Date | string
}

export interface UserGetUserReplyRequestData {
  id: number
  ridArray: number[]
}

export interface UserComment {
  // Comment id → deep-link to it (/topic/:id?comment=<id>).
  id: number
  topicId: number
  content: string
  created: Date | string
}

export interface UserGetUserCommentRequestData {
  id: number
  cidArray: number[]
}

export interface UserFloatingCard extends KunUser {
  moemoepoint: number
  topicCount: number
  topicReplyCount: number
  topicCommentCount: number
  galgameResourceCount: number
}

export type UserUpdateAvatarResponseData = {
  avatar: string
  avatarMin: string
}

export type UserGetUserEmailResponseData = {
  email: string
}
