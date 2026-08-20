import type { z } from 'zod'
import type { updateUpdateLogSchema } from '~/validations/update-log'
import type { createTodoSchema, updateTodoSchema } from '~/validations/todo'

type UpdateUpdateLogPayload = z.infer<typeof updateUpdateLogSchema>
type CreateTodoPayload = z.infer<typeof createTodoSchema>
type UpdateTodoPayload = z.infer<typeof updateTodoSchema>
