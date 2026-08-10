import { z } from 'zod'

export default defineNuxtPlugin(() => {
  z.config(z.locales.zhCN())
})
