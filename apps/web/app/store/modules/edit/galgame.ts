import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import { createEmptyLocaleMap, resetReactiveState } from '~/store/index'
import type { GalgameStorePersist } from '~/store/types/edit/galgame'

export const usePersistEditGalgameStore = defineStore(
  'KUNGalgameEditGalgame',
  () => {
    const vndb_id = ref<GalgameStorePersist['vndb_id']>('')
    const name = reactive<GalgameStorePersist['name']>({
      'en-us': '',
      'ja-jp': '',
      'zh-cn': '',
      'zh-tw': ''
    })
    const introduction = reactive<GalgameStorePersist['introduction']>({
      'en-us': '',
      'ja-jp': '',
      'zh-cn': '',
      'zh-tw': ''
    })
    const content_limit = ref<GalgameStorePersist['content_limit']>('sfw')
    // Wiki defaults: original_language=ja-jp, age_limit=r18. We default
    // age_limit to 'all' instead, because publishing R18 without the user
    // opting in is a content-policy risk on a default-SFW site (per audit
    // §10). User must consciously flip to r18 if applicable.
    const age_limit = ref<GalgameStorePersist['age_limit']>('all')
    const original_language =
      ref<GalgameStorePersist['original_language']>('ja-jp')
    const aliases = ref<GalgameStorePersist['aliases']>([])
    // U1: "" = unknown release date; serialized to wire `release_date` (a
    // bare empty string is valid per the schema's "empty OR YYYY-MM-DD"
    // refinement). TBA defaults false; user opts in for未发布 entries.
    const release_date = ref<GalgameStorePersist['release_date']>('')
    const release_date_tba = ref<GalgameStorePersist['release_date_tba']>(false)

    const resetEditGalgameStore = () => {
      vndb_id.value = ''
      resetReactiveState(name, createEmptyLocaleMap())
      resetReactiveState(introduction, createEmptyLocaleMap())
      content_limit.value = 'sfw'
      age_limit.value = 'all'
      original_language.value = 'ja-jp'
      aliases.value = []
      release_date.value = ''
      release_date_tba.value = false
    }

    return {
      vndb_id,
      name,
      introduction,
      content_limit,
      age_limit,
      original_language,
      aliases,
      release_date,
      release_date_tba,

      resetEditGalgameStore
    }
  },
  {
    persist: {
      // v2: the snake_case migration renamed every scalar field (vndbId→vndb_id,
      // contentLimit→content_limit, …). A pre-migration draft would rehydrate its
      // old camelCase keys as dead cruft while the new snake refs stay at their
      // defaults — a confusing half-reset draft. Bump the key so a stale draft is
      // dropped cleanly (fresh draft) rather than resumed half-populated.
      key: 'KUNGalgameEditGalgame:v2',
      storage: piniaPluginPersistedstate.localStorage()
    }
  }
)
