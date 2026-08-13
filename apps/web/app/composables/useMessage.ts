import type { KunMessageType, KunMessagePosition } from '@kungal/ui-vue'
import { infoMessages } from '~/error/kunMessage'

export const useMessage = (
  messageData: number | string,
  type: KunMessageType,
  duration = 3000,
  richText = false,
  position: KunMessagePosition = 'top-center'
) => {
  const resolved =
    typeof messageData === 'string'
      ? messageData
      : (infoMessages[messageData] ?? '')
  return useKunMessage(resolved, type, duration, richText, position)
}
