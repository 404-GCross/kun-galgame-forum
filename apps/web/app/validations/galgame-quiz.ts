import { z } from 'zod'
import {
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_SPOILER_CONST,
  KUN_QUIZ_SORT_FIELD_CONST
} from '~/constants/galgame-quiz'

export const createGalgameQuizSchema = z.object({
  galgame_ids: z
    .array(z.coerce.number<number>().int().min(1).max(9999999))
    .default([]),
  hide_galgame: z.boolean().default(false),
  category: z.enum(KUN_QUIZ_CATEGORY_CONST),
  type: z.enum(KUN_QUIZ_TYPE_CONST),
  difficulty: z.coerce.number<number>().int().min(1).max(10),
  spoiler_level: z.enum(KUN_QUIZ_SPOILER_CONST).default('none'),
  question: z
    .string()
    .min(1, { message: '请填写题干' })
    .max(200, { message: '题干长度不能超过 200 字' }),
  description: z
    .string()
    .max(20000, { message: '描述长度不能超过 20000 字' })
    .default(''),
  content: z.any(),
  explanation: z
    .string()
    .max(2000, { message: '解析长度不能超过 2000 字' })
    .default('')
})

export const answerGalgameQuizSchema = z.object({
  quiz_id: z.coerce.number<number>().min(1).max(9999999),
  submitted: z.any()
})

export const rateGalgameQuizQualitySchema = z.object({
  quiz_id: z.coerce.number<number>().min(1).max(9999999),
  quality_rating: z.coerce.number<number>().int().min(1).max(10)
})
