const paths = {
  arrow: ['M4 12h16', 'm14 6 6 6-6 6'],
  back: ['M20 12H4', 'm10 6-6 6 6 6'],
  search: ['M21 21l-5-5', 'M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0'],
  edit: ['m16 3 5 5-12 12-6 1 1-6Z', 'm14 5 5 5'],
  book: ['M12 5v16', 'M12 5C8 2 4 3 2 4v15c4-2 7-1 10 2 3-3 6-4 10-2V4c-2-1-6-2-10 1'],
  code: ['m8 6-6 6 6 6', 'm16 6 6 6-6 6', 'm14 3-4 18'],
  folder: ['M3 7V4h7l2 3h9v13H3Z'],
  comment: ['M21 11a9 9 0 0 1-9 9H3l2-5A9 9 0 1 1 21 11Z', 'M8 10h8', 'M8 14h5'],
  check: ['m5 12 4 4L19 6'],
  close: ['m6 6 12 12', 'M18 6 6 18'],
  clock: ['M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0', 'M12 7v5l3 2'],
  plus: ['M12 5v14', 'M5 12h14'],
  heart: ['M20 4c-3-2-6 0-8 2-2-2-5-4-8-2-5 4 0 10 8 16 8-6 13-12 8-16Z'],
  trash: ['M3 6h18', 'M8 6V3h8v3', 'm5 6 1 15h12l1-15', 'M10 10v7', 'M14 10v7'],
  eye: ['M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12', 'M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0'],
  logout: ['M9 4H3v16h6', 'M9 12h12', 'm17 8 4 4-4 4'],
  sun: ['M17 12a5 5 0 1 1-10 0 5 5 0 0 1 10 0', 'M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5'],
  moon: ['M21 13a9 9 0 0 1-10-10 9 9 0 1 0 10 10Z'],
} as const

export default function UiIcon({ name, size = 20 }: { name: keyof typeof paths; size?: number }) {
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    {paths[name].map((path, index) => <path key={index} d={path} />)}
  </svg>
}
