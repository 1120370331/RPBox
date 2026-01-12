// TRP3 Profile 类型定义

export interface Profile {
  id: string
  profileName: string
  characteristics: Characteristics
  about: About
  character: Character
}

export interface Characteristics {
  version: number
  firstName?: string
  lastName?: string
  title?: string
  fullTitle?: string
  race?: string
  class?: string
  classColor?: string
  age?: string
  eyeColor?: string
  eyeColorHex?: string
  height?: string
  weight?: string
  birthplace?: string
  residence?: string
  residenceCoords?: ResidenceCoords
  relationshipStatus?: number
  icon?: string
  background?: number
  miscInfo: MiscInfo[]
  personalityTraits: PersonalityTrait[]
}

export interface ResidenceCoords {
  mapId: number
  x: number
  y: number
  zoneName: string
}

export interface MiscInfo {
  presetType?: number
  name: string
  value: string
  icon: string
}

export interface PersonalityTrait {
  presetId?: number
  leftTrait?: string
  rightTrait?: string
  leftIcon?: string
  rightIcon?: string
  leftColor?: RGBColor
  rightColor?: RGBColor
  value: number
}

export interface RGBColor {
  r: number
  g: number
  b: number
}

export interface About {
  version: number
  template: number
  background?: number
  music?: number
  template1?: Template1
  template2?: Template2Item[]
  template3?: Template3
}

export interface Template1 {
  text: string
}

export interface Template2Item {
  text: string
  icon: string
  background?: number
}

export interface Template3 {
  physical?: TemplateSection
  personality?: TemplateSection
  history?: TemplateSection
}

export interface TemplateSection {
  text: string
  icon?: string
  background?: number
}

export interface Character {
  version: number
  rpStatus: number
  walkUp: number
  currently?: string
  currentlyOOC?: string
}
