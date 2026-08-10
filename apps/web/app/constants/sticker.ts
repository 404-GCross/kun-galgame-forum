const STICKER_BASE = 'https://sticker.kungal.com/stickers'
const SET_SIZES = [80, 80, 80, 80, 80, 80, 18]

export const stickerArray: string[] = SET_SIZES.flatMap((size, setIndex) =>
  Array.from(
    { length: size },
    (_, i) => `${STICKER_BASE}/KUNgal${setIndex + 1}/${i + 1}.webp`
  )
)
