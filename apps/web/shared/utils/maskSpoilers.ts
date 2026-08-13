export const maskSpoilers = (text: string, mask = '███') =>
  text ? text.replace(/\|\|(.*?)\|\|/g, mask) : ''
