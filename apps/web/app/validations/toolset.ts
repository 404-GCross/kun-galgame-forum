import { z } from 'zod'
import {
  KUN_TOOLSET_TYPE_CONST,
  KUN_TOOLSET_LANGUAGE_CONST,
  KUN_TOOLSET_PLATFORM_CONST,
  KUN_TOOLSET_VERSION_CONST
} from '~/constants/toolset'

export const getToolsetSchema = z.object({
  page: z.coerce.number<number>().min(1).max(9999999),
  limit: z.coerce.number<number>().min(1).max(24),
  type: z.enum([...KUN_TOOLSET_TYPE_CONST, 'all']),
  language: z.enum([...KUN_TOOLSET_LANGUAGE_CONST, 'all']),
  platform: z.enum([...KUN_TOOLSET_PLATFORM_CONST, 'all']),
  version: z.enum([...KUN_TOOLSET_VERSION_CONST, 'all']),
  sortField: z.enum(['resource_update_time', 'created', 'view']),
  sortOrder: z.enum(['asc', 'desc'])
})

export const getToolsetDetailSchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999)
})

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
    toolsetId: z.coerce.number<number>().min(1).max(9999999)
  })
)

export const deleteToolsetDetailSchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999)
})

export const getToolsetPracticalitySchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999)
})

export const updateToolsetPracticalitySchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999),
  rate: z.coerce.number<number>().min(1).max(5)
})

// Toolset comment
export const getToolsetCommentSchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999),
  page: z.coerce.number<number>().min(1).max(9999999),
  limit: z.coerce.number<number>().min(1).max(30),
  sortOrder: z.enum(['asc', 'desc'])
})

export const createToolsetCommentSchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999),
  content: z.string().min(1).max(1007),
  parentId: z.coerce.number<number>().min(1).optional().nullable()
})

export const updateToolsetCommentSchema = z.object({
  commentId: z.coerce.number<number>().min(1).max(9999999),
  content: z.string().min(1).max(1007)
})

export const deleteToolsetCommentSchema = z.object({
  commentId: z.coerce.number<number>().min(1).max(9999999)
})

// Toolset resource & upload
export const getToolsetResourceDetailSchema = z.object({
  toolsetResourceId: z.coerce.number<number>().min(1).max(9999999)
})

// In s3 mode the wire value of `size` is a raw byte-count string
// ("1572864") — the resource list later parses it back via Number(...).
// In user mode it must be a human-readable "1007MB" / "0721GB" because
// the user types it themselves and Item.vue prints it verbatim.
export const createToolsetResourceSchema = z
  .object({
    toolsetId: z.coerce.number<number>().min(1).max(9999999),
    type: z.enum(['s3', 'user']),
    content: z.string().max(1007).optional().default(''),
    // s3 resources carry the completed-upload artifact uuid (the download URL is
    // resolved server-side from it); content stays empty.
    artifactUuid: z.string().max(36).optional().default(''),
    size: z.string(),
    code: z.string().max(1007).optional().default(''),
    password: z.string().max(1007).optional().default(''),
    note: z.string().max(1007).optional().default('')
  })
  .superRefine((val, ctx) => {
    if (val.type === 's3') {
      if (!val.artifactUuid) {
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

// Same byte-vs-formatted contract as createToolsetResourceSchema: in s3
// mode `size` is a raw byte-count string (matches what the resource row
// already stores), in user mode it's a kb/mb/gb-suffixed human string.
// The s3 branch is mostly a guard for hygiene — the API ignores size
// updates on s3 rows anyway — but rejecting impossible inputs here keeps
// the schema honest and avoids silent acceptance of typo'd values.
export const updateToolsetResourceSchema = z
  .object({
    toolsetResourceId: z.coerce.number<number>().min(1).max(9999999),
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

export const deleteToolsetResourceSchema = z.object({
  toolsetResourceId: z.coerce.number<number>().min(1).max(9999999)
})

export const initToolsetUploadSchema = z.object({
  toolsetId: z.coerce.number<number>().min(1).max(9999999),
  filename: z
    .string()
    .min(1)
    .max(1007)
    .regex(/\.(7z|zip|rar)$/i, {
      message: '文件名必须以 .7z, .zip 或 .rar 结尾'
    }),
  filesize: z.coerce.number<number>().int().positive(),
  contentType: z.string().min(1).max(100)
})

export const completeToolsetUploadSchema = z.object({
  artifactUuid: z.string().min(1).max(36),
  parts: z
    .array(z.object({ partNumber: z.number().int().min(1), etag: z.string() }))
    .optional()
})

export const resumeToolsetUploadSchema = z.object({
  artifactUuid: z.string().min(1).max(36)
})

export const abortToolsetUploadSchema = z.object({
  artifactUuid: z.string().min(1).max(36)
})
