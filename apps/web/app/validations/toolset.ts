import { z } from 'zod'
import {
  KUN_TOOLSET_TYPE_CONST,
  KUN_TOOLSET_LANGUAGE_CONST,
  KUN_TOOLSET_PLATFORM_CONST,
  KUN_TOOLSET_VERSION_CONST
} from '~/constants/toolset'

export const createToolsetSchema = z.object({
  name: z.string().min(1).max(500),
  description: z.string().max(2000).default(''),
  language: z.enum(KUN_TOOLSET_LANGUAGE_CONST, { message: '非法的语言' }),
  platform: z.enum(KUN_TOOLSET_PLATFORM_CONST, { message: '非法的平台' }),
  type: z.enum(KUN_TOOLSET_TYPE_CONST, { message: '非法的工具类型' }),
  version: z.enum(KUN_TOOLSET_VERSION_CONST, { message: '非法的版本类型' }),
  homepage: z.array(z.url().max(500)).default([]),
  aliases: z.array(z.string().min(1).max(500)).default([])
})

export const updateToolsetSchema = createToolsetSchema.merge(
  z.object({
    toolset_id: z.coerce.number<number>().min(1).max(9999999)
  })
)

export const createToolsetResourceSchema = z
  .object({
    toolset_id: z.coerce.number<number>().min(1).max(9999999),
    type: z.enum(['s3', 'user']),
    content: z.string().max(1007).optional().default(''),
    artifact_uuid: z.string().max(36).optional().default(''),
    size: z.string(),
    code: z.string().max(1007).optional().default(''),
    password: z.string().max(1007).optional().default(''),
    note: z.string().max(1007).optional().default('')
  })
  .superRefine((val, ctx) => {
    if (val.type === 's3') {
      if (!val.artifact_uuid) {
        ctx.addIssue({
          code: 'custom',
          path: ['artifactUuid'],
          message: '请先上传文件'
        })
      }
      if (!/^\d+$/.test(val.size)) {
        ctx.addIssue({
          code: 'custom',
          path: ['size'],
          message: 's3 资源的大小必须是字节数'
        })
      }
    } else if (!ResourceSizePattern.test(val.size)) {
      ctx.addIssue({
        code: 'custom',
        path: ['size'],
        message: '大小格式不正确, 需要包含 KB, MB, GB'
      })
    }
  })

export const updateToolsetResourceSchema = z
  .object({
    toolset_resource_id: z.coerce.number<number>().min(1).max(9999999),
    type: z.enum(['s3', 'user']),
    content: z.string().max(1007).optional().default(''),
    size: z.string(),
    code: z.string().max(1007).optional().default(''),
    password: z.string().max(1007).optional().default(''),
    note: z.string().max(1007).optional().default('')
  })
  .superRefine((val, ctx) => {
    if (val.type === 's3') {
      if (!/^\d+$/.test(val.size)) {
        ctx.addIssue({
          code: 'custom',
          path: ['size'],
          message: 's3 资源的大小必须是字节数'
        })
      }
    } else if (!ResourceSizePattern.test(val.size)) {
      ctx.addIssue({
        code: 'custom',
        path: ['size'],
        message: '大小格式不正确, 需要包含 KB, MB, GB'
      })
    }
  })

export const initToolsetUploadSchema = z.object({
  toolset_id: z.coerce.number<number>().min(1).max(9999999),
  filename: z
    .string()
    .min(1)
    .max(1007)
    .regex(/\.(7z|zip|rar)$/i, {
      message: '文件名必须以 .7z, .zip 或 .rar 结尾'
    }),
  filesize: z.coerce.number<number>().int().positive(),
  content_type: z.string().min(1).max(100)
})

export const completeToolsetUploadSchema = z.object({
  artifact_uuid: z.string().min(1).max(36),
  parts: z
    .array(z.object({ part_number: z.number().int().min(1), etag: z.string() }))
    .optional()
})

export const resumeToolsetUploadSchema = z.object({
  artifact_uuid: z.string().min(1).max(36)
})

export const abortToolsetUploadSchema = z.object({
  artifact_uuid: z.string().min(1).max(36)
})
