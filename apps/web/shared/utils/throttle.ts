// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function throttle<T extends (...args: any[]) => void>(
  executeCallback: T,
  delay: number,
  delayedCallback?: (...args: Parameters<T>) => unknown
) {
  let lastExecution = 0
  let timeout: NodeJS.Timeout | null = null

  const throttled = (...args: Parameters<T>) => {
    const now = Date.now()

    if (!lastExecution || now - lastExecution >= delay) {
      executeCallback(...args)
      lastExecution = now
    } else if (!timeout && delayedCallback) {
      delayedCallback(...args)
      timeout = setTimeout(() => {
        timeout = null
      }, delay)
    }
  }

  return throttled as (...args: Parameters<T>) => void
}
