<script setup lang="ts">
// "查看所有封面" modal for the galgame detail banner.
//
// Unlike moyu's patch header (a whitelist DTO without the covers array), the
// forum's galgame detail payload ALREADY carries the full covers[] — so the
// modal just groups + renders what it's given, no extra fetch.
//
// The wiki syncs a VN's whole VNDB /cv gallery (main + every release cover),
// each tagged with `kind`; we group by kind so 主封面 / 盒装正面 / 数字版 / 封底 …
// are separated. Each cover is sized to its REAL aspect ratio (no crop, no
// letterbox bars) with a ThumbHash blur-up; the lightbox opens the full image.
//
// Best-cover voting (wave 176): each cover carries the catalog's advisory vote tally
// and this viewer's ballot. ONE ballot per game — voting another cover MOVES
// it, which is why a successful vote clears every other cover's voted flag
// locally instead of only flipping the one clicked. The vote is the forum's
// first write that travels as the USER's own OAuth token, so a session
// authorized before the `catalog:edit` scope existed is refused with code 235
// and told to log out and back in (kunFetch surfaces that message).
const props = defineProps<{ gid: number; covers: GalgameCover[] }>()
const open = defineModel<boolean>({ required: true })

// Kind → display label, in the order sections should appear. Anything unknown /
// empty falls into 其它 at the end.
const KIND_LABEL: Record<string, string> = {
  main: '主封面',
  pkgfront: '盒装正面',
  dig: '数字版',
  pkgback: '封底',
  pkgcontent: '内页',
  pkgside: '书脊',
  pkgmed: '碟面',
  '': '其它'
}
const KIND_ORDER = Object.keys(KIND_LABEL)

// A local copy of the tallies, keyed by the catalog cover id: the covers prop
// belongs to the detail payload and a vote must not mutate it, but the counts
// have to move the moment the server answers.
interface CoverBallot {
  count: number
  voted: boolean
}
const ballots = ref(new Map<number, CoverBallot>())

const seedBallots = () => {
  const seeded = new Map<number, CoverBallot>()
  for (const c of props.covers) {
    if (c.id) {
      seeded.set(c.id, { count: c.vote_count ?? 0, voted: !!c.voted })
    }
  }
  ballots.value = seeded
}
seedBallots()
watch(() => props.covers, seedBallots)

const ballotOf = (cover: GalgameCover): CoverBallot | undefined =>
  cover.id ? ballots.value.get(cover.id) : undefined

// The cover id currently in flight (0 = none): one ballot per game, so votes
// are serialized rather than queued per cover.
const voting = ref(0)

// A vote MOVES the single ballot, so the map is rewritten from the server's
// answer for the clicked cover: it becomes the only voted one, and whichever
// cover held the ballot before loses its count.
const applyVote = (coverId: number, count: number, voted: boolean) => {
  const next = new Map(ballots.value)
  for (const [id, ballot] of next) {
    if (id === coverId) continue
    if (voted && ballot.voted) {
      next.set(id, { count: Math.max(0, ballot.count - 1), voted: false })
    }
  }
  next.set(coverId, { count, voted })
  ballots.value = next
}

const toggleVote = async (cover: GalgameCover) => {
  if (!cover.id || voting.value) return
  if (!requireLogin()) return
  const willUnvote = !!ballotOf(cover)?.voted
  voting.value = cover.id
  const result = await kunFetch<{
    cover_id: number
    vote_count: number
    voted: boolean
  }>(`/galgame/${props.gid}/cover/${cover.id}/vote`, {
    method: willUnvote ? 'DELETE' : 'PUT'
  })
  voting.value = 0
  if (!result) {
    return
  }
  applyVote(result.cover_id, result.vote_count, result.voted)
}

const sorted = computed(() =>
  [...props.covers]
    .filter((c) => !!c.image_hash)
    .sort((a, b) => a.sort_order - b.sort_order)
)

// Covers grouped into ordered, labeled sections (only non-empty kinds shown).
const groups = computed(() => {
  const byKind = new Map<string, GalgameCover[]>()
  for (const c of sorted.value) {
    const k = KIND_LABEL[c.kind ?? ''] !== undefined ? (c.kind ?? '') : ''
    if (!byKind.has(k)) byKind.set(k, [])
    byKind.get(k)!.push(c)
  }
  return KIND_ORDER.filter((k) => byKind.has(k)).map((k) => ({
    kind: k,
    label: KIND_LABEL[k],
    covers: byKind.get(k)!
  }))
})
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-3xl w-full">
    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <div class="bg-primary h-6 w-1 rounded" />
        <h2 class="text-xl font-bold">所有封面</h2>
      </div>

      <KunNull v-if="!sorted.length" description="该 Galgame 暂无封面" />

      <KunLightboxGallery v-else>
        <div class="space-y-5">
          <section v-for="g in groups" :key="g.kind" class="space-y-2">
            <h3 class="text-default-600 text-sm font-medium">
              {{ g.label }}
              <span class="text-default-400">({{ g.covers.length }})</span>
            </h3>
            <!-- items-start: covers have varied real aspect ratios, so the grid
                 must NOT stretch a row's cells to equal height — otherwise a
                 short (landscape) cell shows its figure background as bars next
                 to a tall (portrait) neighbour. Top-align so each figure hugs
                 its own cover. -->
            <div class="grid grid-cols-1 items-start gap-3 sm:grid-cols-2">
              <div
                v-for="c in g.covers"
                :key="c.image_hash"
                class="space-y-1.5"
              >
                <KunLightboxGalleryItem
                  :src="galgameImageSrc(c)"
                  :alt="g.label"
                  as="figure"
                  class="border-default/20 bg-default-100 block overflow-hidden rounded-lg border"
                >
                  <!-- Box sized to the cover's REAL aspect ratio + full image with
                       object-cover, so box ratio == image ratio: no crop AND no
                       bars. Pre-backfill the ratio falls back to 16/9. Few covers,
                       opt-in modal, lightbox loads full on click — so serving full
                       here costs little. -->
                  <KunImage
                    :src="galgameImageSrc(c)"
                    :alt="g.label"
                    loading="lazy"
                    :aspect-ratio="imageAspectRatio(c.width, c.height)"
                    :thumbhash="c.thumbhash"
                    class-name="bg-default-100"
                  />
                </KunLightboxGalleryItem>

                <!-- The vote control is a SIBLING of the lightbox item, never a
                     child: nested, every click would also open the lightbox. A
                     cover the catalog did not name (no id) gets none — there is
                     nothing to address the vote to. -->
                <KunButton
                  v-if="c.id"
                  variant="flat"
                  size="sm"
                  :color="ballotOf(c)?.voted ? 'primary' : 'default'"
                  :loading="voting === c.id"
                  full-width
                  @click="toggleVote(c)"
                >
                  <KunIcon
                    :name="ballotOf(c)?.voted ? 'lucide:heart' : 'lucide:plus'"
                    class="size-4"
                  />
                  {{ ballotOf(c)?.voted ? '已选为最佳封面' : '选为最佳封面' }}
                  <span class="text-default-500">
                    {{ ballotOf(c)?.count ?? 0 }}
                  </span>
                </KunButton>
              </div>
            </div>
          </section>
        </div>
      </KunLightboxGallery>
    </div>
  </KunModal>
</template>
