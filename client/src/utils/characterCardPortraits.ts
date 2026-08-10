import type { CharacterCardPortraitImage, CharacterCardSummary } from '@/api/characterCard'

export function normalizeCharacterCardPortraits(
  card?: Pick<CharacterCardSummary, 'portrait_image_url' | 'portrait_image_updated_at' | 'portraits'> | null,
): CharacterCardPortraitImage[] {
  const portraits = Array.isArray(card?.portraits)
    ? card.portraits
      .filter((portrait) => portrait && Number.isFinite(Number(portrait.id)) && Boolean(portrait.image_url))
      .map((portrait) => ({
        id: Number(portrait.id),
        image_url: portrait.image_url,
        image_updated_at: portrait.image_updated_at || null,
        sort_order: Number.isFinite(Number(portrait.sort_order)) ? Number(portrait.sort_order) : 0,
        is_cover: portrait.is_cover === true,
      }))
      .sort((left, right) => left.sort_order - right.sort_order || left.id - right.id)
    : []

  if (portraits.length) {
    if (!portraits.some((portrait) => portrait.is_cover)) portraits[0].is_cover = true
    return portraits
  }

  if (!card?.portrait_image_url) return []
  return [{
    id: 0,
    image_url: card.portrait_image_url,
    image_updated_at: card.portrait_image_updated_at || null,
    sort_order: 0,
    is_cover: true,
  }]
}

export function getCharacterCardCoverPortrait(
  portraits: CharacterCardPortraitImage[],
): CharacterCardPortraitImage | null {
  return portraits.find((portrait) => portrait.is_cover) || portraits[0] || null
}
