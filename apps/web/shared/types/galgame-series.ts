// The PUBLIC series browse types retired with the wiki series vocabulary (P3):
// 146 wiki series, only 6 of which correspond to anything in the catalog. What
// remains here serves the STAFF editor only — the wiki rows live on until the
// editing engine retires that face, and the admin console still curates them.

// Wiki /series/search and /series/modal return FULL galgame rows
// (snake_case multi-language `name_<locale>` columns plus a bunch of
// other fields the select widget doesn't need). The widget reads
// `id` for the value and runs the names through `galgameNameFromWire`
// to pick the user-preferred locale. Extra wire fields are tolerated
// but unused.
export interface GalgameSeriesSearchItem {
  id: number
  name_en_us?: string
  name_ja_jp?: string
  name_zh_cn?: string
  name_zh_tw?: string
}
