export const mobileFeatures = ['community', 'stories', 'market', 'guild', 'profile', 'profiles'] as const

export type MobileFeature = (typeof mobileFeatures)[number]

export function isFeatureEnabled(feature: string): feature is MobileFeature {
  return (mobileFeatures as readonly string[]).includes(feature)
}
