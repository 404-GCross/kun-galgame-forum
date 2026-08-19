import { z } from 'zod'
import { KUN_UPDATE_LOG } from '~/constants/update'

export const createUpdateLogSchema = z.object({
  version: z.string().min(1).max(20, '更新版本号最多 20 个字符'),
  content: z
    .string()
    .max(1000, '更新描述最多 1000 个字符')
    .optional()
    .default(''),
  type: z.enum(KUN_UPDATE_LOG)
})

export const updateUpdateLogSchema = createUpdateLogSchema.extend({
  update_log_id: z.coerce.number<number>().min(1).max(9999999)
})
