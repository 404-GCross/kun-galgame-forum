import { z } from 'zod'
import { KUN_TODO_TYPE_CONST } from '~/constants/update'

export const createTodoSchema = z.object({
  type: z.enum(KUN_TODO_TYPE_CONST),
  content: z.string().max(1000, '待办最多 1000 个字符').optional().default('')
})

export const updateTodoSchema = z.object({
  todo_id: z.coerce.number<number>().min(1).max(9999999),
  type: z.enum(KUN_TODO_TYPE_CONST),
  content: z.string().max(1000, '待办最多 1000 个字符').optional().default('')
})
