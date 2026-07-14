export interface TomTomWaypointLike {
  map_id?: string
  zone?: string
  x?: number
  y?: number
  label?: string
  title?: string
}

export interface ParsedTomTomWaypoint {
  map_id: string
  zone: string
  x: number
  y: number
  label: string
}

export interface TomTomCommandOptions {
  sequence?: number
  total?: number
}

function parseCoordinate(value?: string) {
  if (!value?.trim()) return undefined
  const coordinate = Number(value.replace(',', '.'))
  if (!Number.isFinite(coordinate) || coordinate < 0 || coordinate > 100) return undefined
  return coordinate
}

function cleanLabel(value?: string) {
  return String(value || '').replace(/\s+/g, ' ').trim()
}

function normalizeMapID(value?: string) {
  const mapID = String(value || '').trim()
  if (!mapID) return ''
  return /^#?\d+$/.test(mapID) ? mapID.replace(/^#/, '') : mapID
}

export function hasTomTomCoordinates(waypoint: TomTomWaypointLike) {
  const x = Number(waypoint.x)
  const y = Number(waypoint.y)
  return Number.isFinite(x)
    && Number.isFinite(y)
    && x >= 0
    && x <= 100
    && y >= 0
    && y <= 100
    && (x !== 0 || y !== 0)
}

export function formatTomTomCommand(waypoint: TomTomWaypointLike, options: TomTomCommandOptions = {}) {
  if (!hasTomTomCoordinates(waypoint)) return ''

  const mapID = normalizeMapID(waypoint.map_id)
  const target = mapID
    ? (/^\d+$/.test(mapID) ? `#${mapID}` : mapID)
    : cleanLabel(waypoint.zone)
  const routePrefix = options.total && options.total > 1 && options.sequence
    ? `[${options.sequence}/${options.total}] `
    : ''
  const fallbackLabel = routePrefix ? `路线点 ${options.sequence}` : ''
  const label = `${routePrefix}${cleanLabel(waypoint.label || waypoint.title) || fallbackLabel}`.trim()
  const parts = [
    '/way',
    target,
    Number(waypoint.x).toFixed(2),
    Number(waypoint.y).toFixed(2),
    label,
  ]
  return parts.filter(Boolean).join(' ')
}

export function parseTomTomCommand(line: string): ParsedTomTomWaypoint | undefined {
  const command = line.trim().match(/^\/(?:way|tway|tomtomway)\s+(.+)$/i)
  if (!command) return undefined

  const tokens = command[1].trim().split(/\s+/)
  if (tokens.length < 2) return undefined

  let mapID = ''
  let zone = ''
  let coordinateIndex = 0

  if (/^#\d+$/.test(tokens[0])) {
    mapID = tokens[0].slice(1)
    coordinateIndex = 1
  } else if (Number.isFinite(Number(tokens[0])) && parseCoordinate(tokens[1]) !== undefined) {
    const legacyMapID = /^\d+$/.test(tokens[0])
      && parseCoordinate(tokens[2]) !== undefined
      && tokens.length >= 4
    if (legacyMapID) {
      mapID = tokens[0]
      coordinateIndex = 1
    } else if (parseCoordinate(tokens[0]) === undefined) {
      return undefined
    }
  } else {
    coordinateIndex = tokens.findIndex(token => parseCoordinate(token) !== undefined)
    if (coordinateIndex <= 0) return undefined
    zone = tokens.slice(0, coordinateIndex).join(' ')
  }

  const x = parseCoordinate(tokens[coordinateIndex])
  const y = parseCoordinate(tokens[coordinateIndex + 1])
  if (x === undefined || y === undefined || (x === 0 && y === 0)) return undefined

  return {
    map_id: mapID,
    zone,
    x,
    y,
    label: cleanLabel(tokens.slice(coordinateIndex + 2).join(' ')),
  }
}

export function parseTomTomCommands(value: string) {
  const waypoints: ParsedTomTomWaypoint[] = []
  const rejected: string[] = []

  value.split(/\r?\n/).map(line => line.trim()).filter(Boolean).forEach((line) => {
    const waypoint = parseTomTomCommand(line)
    if (waypoint) waypoints.push(waypoint)
    else rejected.push(line)
  })

  return { waypoints, rejected }
}
