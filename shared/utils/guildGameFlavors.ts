export type GuildGameFlavor = '_retail_' | '_forever_'

/** Older guilds only have game_flavor; keep their version selection visible. */
export function guildGameFlavors(guild: {
  game_flavors?: GuildGameFlavor[] | null
  game_flavor?: GuildGameFlavor | null
}): GuildGameFlavor[] {
  const values = guild.game_flavors?.length ? guild.game_flavors : [guild.game_flavor || '_retail_']
  return (['_retail_', '_forever_'] as GuildGameFlavor[]).filter(flavor => values.includes(flavor))
}
