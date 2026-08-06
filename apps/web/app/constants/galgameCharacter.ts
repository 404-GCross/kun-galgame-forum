// Billing on the 登场角色 roster. `unknown` is deliberately ABSENT: it is a
// meaningful catalog value (a source that publishes no billing, or a character
// reached only through a voice credit), but "未知" is not worth a chip — a miss
// here renders no badge at all, exactly like the staff gender table.
export const KUN_GALGAME_CHARACTER_KIND_MAP: Record<string, string> = {
  main: '主角',
  secondary: '配角',
  appears: '登场'
}

// Only the main cast gets a coloured badge; 配角 / 登场 are ordinary. The roster
// arrives sorted main-first, so the colour is reinforcing the order rather than
// carrying it.
export const KUN_GALGAME_CHARACTER_KIND_COLOR: Record<
  string,
  'primary' | 'default'
> = {
  main: 'primary'
}

// How much a character's mere presence gives away (VNDB `chars_vns.spoil`).
// Anything above 0 is withheld behind an explicit click.
export const KUN_GALGAME_CHARACTER_SPOILER_MAP: Record<number, string> = {
  1: '轻微剧透',
  2: '严重剧透'
}
