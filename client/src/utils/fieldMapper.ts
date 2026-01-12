import type {
  Profile,
  Characteristics,
  About,
  Character,
  MiscInfo,
  PersonalityTrait,
  Template3,
} from './types'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type RawData = Record<string, any>

export function mapProfile(raw: RawData, id: string): Profile {
  const player = raw.player || {}
  return {
    id,
    profileName: raw.profileName || '未命名',
    characteristics: mapCharacteristics(player.characteristics || {}),
    about: mapAbout(player.about || {}),
    character: mapCharacter(player.character || {}),
  }
}

export function mapCharacteristics(raw: RawData): Characteristics {
  return {
    version: raw.v || 1,
    firstName: raw.FN,
    lastName: raw.LN,
    title: raw.TI,
    fullTitle: raw.FT,
    race: raw.RA,
    class: raw.CL,
    classColor: raw.CH,
    age: raw.AG,
    eyeColor: raw.EC,
    eyeColorHex: raw.EH,
    height: raw.HE,
    weight: raw.WE,
    birthplace: raw.BP,
    residence: raw.RE,
    residenceCoords: raw.RC ? {
      mapId: raw.RC[0],
      x: raw.RC[1],
      y: raw.RC[2],
      zoneName: raw.RC[3],
    } : undefined,
    relationshipStatus: raw.RS,
    icon: raw.IC,
    background: raw.bkg,
    miscInfo: mapMiscInfo(raw.MI || []),
    personalityTraits: mapPersonalityTraits(raw.PS || []),
  }
}

function mapMiscInfo(raw: RawData[] | RawData): MiscInfo[] {
  if (!Array.isArray(raw)) return []
  return raw.map(item => ({
    presetType: item.ID,
    name: item.NA || '',
    value: item.VA || '',
    icon: item.IC || '',
  }))
}

function mapPersonalityTraits(raw: RawData[] | RawData): PersonalityTrait[] {
  if (!Array.isArray(raw)) return []
  return raw.map(item => ({
    presetId: item.ID,
    leftTrait: item.LT,
    rightTrait: item.RT,
    leftIcon: item.LI,
    rightIcon: item.RI,
    leftColor: item.LC,
    rightColor: item.RC,
    value: item.V2 || 10,
  }))
}

export function mapAbout(raw: RawData): About {
  const result: About = {
    version: raw.v || 1,
    template: raw.TE || 1,
    background: raw.BK,
    music: raw.MU,
  }

  if (raw.T1) {
    result.template1 = { text: raw.T1.TX || '' }
  }

  if (raw.T2 && Array.isArray(raw.T2)) {
    result.template2 = raw.T2.map((item: RawData) => ({
      text: item.TX || '',
      icon: item.IC || '',
      background: item.BK,
    }))
  }

  if (raw.T3) {
    result.template3 = mapTemplate3(raw.T3)
  }

  return result
}

function mapTemplate3(raw: RawData): Template3 {
  const result: Template3 = {}

  if (raw.PH) {
    result.physical = {
      text: raw.PH.TX || '',
      icon: raw.PH.IC,
      background: raw.PH.BK,
    }
  }

  if (raw.PS) {
    result.personality = {
      text: raw.PS.TX || '',
      icon: raw.PS.IC,
      background: raw.PS.BK,
    }
  }

  if (raw.HI) {
    result.history = {
      text: raw.HI.TX || '',
      icon: raw.HI.IC,
      background: raw.HI.BK,
    }
  }

  return result
}

function mapCharacter(raw: RawData): Character {
  return {
    version: raw.v || 1,
    rpStatus: raw.RP || 1,
    walkUp: raw.WU || 1,
    currently: raw.CU,
    currentlyOOC: raw.CO,
  }
}

// 反向映射函数
export function unmapProfile(profile: Profile): RawData {
  return {
    profileName: profile.profileName,
    player: {
      characteristics: unmapCharacteristics(profile.characteristics),
      about: unmapAbout(profile.about),
      character: unmapCharacter(profile.character),
    },
  }
}

function unmapCharacteristics(c: Characteristics): RawData {
  const result: RawData = { v: c.version }

  if (c.firstName) result.FN = c.firstName
  if (c.lastName) result.LN = c.lastName
  if (c.title) result.TI = c.title
  if (c.fullTitle) result.FT = c.fullTitle
  if (c.race) result.RA = c.race
  if (c.class) result.CL = c.class
  if (c.classColor) result.CH = c.classColor
  if (c.age) result.AG = c.age
  if (c.eyeColor) result.EC = c.eyeColor
  if (c.eyeColorHex) result.EH = c.eyeColorHex
  if (c.height) result.HE = c.height
  if (c.weight) result.WE = c.weight
  if (c.birthplace) result.BP = c.birthplace
  if (c.residence) result.RE = c.residence
  if (c.residenceCoords) {
    result.RC = [
      c.residenceCoords.mapId,
      c.residenceCoords.x,
      c.residenceCoords.y,
      c.residenceCoords.zoneName,
    ]
  }
  if (c.relationshipStatus !== undefined) result.RS = c.relationshipStatus
  if (c.icon) result.IC = c.icon
  if (c.background !== undefined) result.bkg = c.background

  if (c.miscInfo.length > 0) {
    result.MI = c.miscInfo.map(item => ({
      ID: item.presetType,
      NA: item.name,
      VA: item.value,
      IC: item.icon,
    }))
  }

  if (c.personalityTraits.length > 0) {
    result.PS = c.personalityTraits.map(item => ({
      ID: item.presetId,
      LT: item.leftTrait,
      RT: item.rightTrait,
      LI: item.leftIcon,
      RI: item.rightIcon,
      LC: item.leftColor,
      RC: item.rightColor,
      V2: item.value,
    }))
  }

  return result
}

function unmapAbout(a: About): RawData {
  const result: RawData = {
    v: a.version,
    TE: a.template,
  }

  if (a.background !== undefined) result.BK = a.background
  if (a.music !== undefined) result.MU = a.music

  if (a.template1) {
    result.T1 = { TX: a.template1.text }
  }

  if (a.template2) {
    result.T2 = a.template2.map(item => ({
      TX: item.text,
      IC: item.icon,
      BK: item.background,
    }))
  }

  if (a.template3) {
    result.T3 = {}
    if (a.template3.physical) {
      result.T3.PH = {
        TX: a.template3.physical.text,
        IC: a.template3.physical.icon,
        BK: a.template3.physical.background,
      }
    }
    if (a.template3.personality) {
      result.T3.PS = {
        TX: a.template3.personality.text,
        IC: a.template3.personality.icon,
        BK: a.template3.personality.background,
      }
    }
    if (a.template3.history) {
      result.T3.HI = {
        TX: a.template3.history.text,
        IC: a.template3.history.icon,
        BK: a.template3.history.background,
      }
    }
  }

  return result
}

function unmapCharacter(c: Character): RawData {
  return {
    v: c.version,
    RP: c.rpStatus,
    WU: c.walkUp,
    CU: c.currently,
    CO: c.currentlyOOC,
  }
}
