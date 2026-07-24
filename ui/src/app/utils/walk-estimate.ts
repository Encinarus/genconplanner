export interface WalkEstimate {
  feet: number;
  minMinutes: number;
  maxMinutes: number;
  displayText: string;
}

export function parseMapLinkParams(mapLink?: string): { lg: number; lt: number; f: number; c: string } | null {
  if (!mapLink) return null;
  try {
    const url = new URL(mapLink, 'https://www.gencon.com');
    const lgStr = url.searchParams.get('lg');
    const ltStr = url.searchParams.get('lt');
    if (!lgStr || !ltStr) return null;

    const lg = parseFloat(lgStr);
    const lt = parseFloat(ltStr);
    const f = parseInt(url.searchParams.get('f') || '0', 10);
    const c = url.searchParams.get('c') || '0';

    if (isNaN(lg) || isNaN(lt)) return null;
    return { lg, lt, f, c };
  } catch {
    return null;
  }
}

export function estimateWalkTimeBetweenMapLinks(
  link1?: string,
  link2?: string
): WalkEstimate | null {
  const p1 = parseMapLinkParams(link1);
  const p2 = parseMapLinkParams(link2);

  if (!p1 || !p2) return null;

  // Check if same location
  if (link1 === link2 || (p1.lg === p2.lg && p1.lt === p2.lt && p1.f === p2.f && p1.c === p2.c)) {
    return {
      feet: 0,
      minMinutes: 0,
      maxMinutes: 0,
      displayText: '0 min walk'
    };
  }

  const manhattanUnits = Math.abs(p2.lg - p1.lg) + Math.abs(p2.lt - p1.lt);
  const feet = Math.round(manhattanUnits * 10.0);

  // Fast speed: 2.5 mph = 220.0 ft/min
  // Slower speed: 1.8 mph = 158.4 ft/min
  let minMin = feet / 220.0;
  let maxMin = feet / 158.4;

  // Floor difference penalty (+1.5m to +2.5m per floor)
  const floorDiff = Math.abs(p2.f - p1.f);
  minMin += floorDiff * 1.5;
  maxMin += floorDiff * 2.5;

  // Stadium Connector Tunnel penalty (+3m to +5m)
  if (p1.c !== p2.c) {
    minMin += 3.0;
    maxMin += 5.0;
  }

  const roundedMin = Math.round(minMin);
  const roundedMax = Math.max(roundedMin, Math.round(maxMin));

  const displayText = (roundedMin === roundedMax)
    ? `${roundedMin} min walk`
    : `${roundedMin}–${roundedMax} min walk`;

  return {
    feet,
    minMinutes: roundedMin,
    maxMinutes: roundedMax,
    displayText
  };
}
