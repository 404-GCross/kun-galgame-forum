export type UpdateType =
  | 'feat'
  | 'pref'
  | 'fix'
  | 'styles'
  | 'mod'
  | 'chore'
  | 'sec'
  | 'refactor'
  | 'docs'
  | 'test'

export interface UpdateTodo {
  id: number
  status: number
  type: string
  content: string
  completed_time: Date | string | null
  user_id: number
  user: KunUser
  claimed_user_id: number | null
  claimed_user?: KunUser | null
  created: Date | string
  updated: Date | string
}

export interface UpdateLog {
  id: number
  type: UpdateType
  version: string
  content: string
  user_id: number
  created: Date | string
  updated: Date | string
}

export interface UpdateHistoryList {
  updates: UpdateLog[]
  total: number
}

export interface UpdateTodoList {
  todos: UpdateTodo[]
  total: number
}
