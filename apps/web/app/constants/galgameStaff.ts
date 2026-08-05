// The registry's gender codes for a PERSON. "Unknown" is the absence of a code
// rather than a code of its own, which is why this table has only two entries:
// a name the lookup misses renders no 性别 row at all instead of 「未知」.
export const KUN_GALGAME_STAFF_GENDER_MAP: Record<number, string> = {
  1: '男',
  2: '女'
}
