const MOSAIC_COLUMNS = 18
const MOSAIC_ROWS = 7

const leatherTones = [
  '#3f281f',
  '#523326',
  '#68412d',
  '#7e5136',
  '#93613f',
  '#a06e49',
]

function isWalletCell(row, column) {
  if (row === 0) {
    return column >= 4 && column <= 13
  }
  if (row === 1) {
    return column >= 2 && column <= 15
  }
  if (row === 2) {
    return column >= 0 && column <= 17
  }
  if (row >= 3 && row <= 5) {
    return column >= 1 && column <= 16
  }
  return column >= 3 && column <= 14
}

function isFlapCell(row) {
  return row <= 2
}

function cellDelay(row, column) {
  const centerDistance = Math.abs(column - 8.5) + Math.abs(row - 3)
  const jitter = ((row * 17 + column * 11) % 9) * 46
  return Math.round(centerDistance * 74 + jitter)
}

const cells = Array.from({ length: MOSAIC_ROWS * MOSAIC_COLUMNS }, (_, index) => {
  const row = Math.floor(index / MOSAIC_COLUMNS)
  const column = index % MOSAIC_COLUMNS
  const finalCell = isWalletCell(row, column)
  const tone = leatherTones[(row * 5 + column * 3) % leatherTones.length]

  return {
    column,
    finalCell,
    index,
    row,
    tone,
    delay: cellDelay(row, column),
    shade: isFlapCell(row) ? 'flap' : 'body',
  }
})

export default function WalletMosaicAnimation() {
  return (
    <div className="mosaic-wallet-stage" aria-hidden="true">
      <span className="mosaic-wallet-shadow" />
      <div className="mosaic-wallet-grid">
        {cells.map((cell) => (
          <span
            className={[
              'mosaic-wallet-cell',
              cell.finalCell ? 'mosaic-wallet-cell-final' : 'mosaic-wallet-cell-noise',
              `mosaic-wallet-cell-${cell.shade}`,
            ].join(' ')}
            key={cell.index}
            style={{
              '--cell-color': cell.tone,
              '--cell-delay': `${cell.delay}ms`,
              '--cell-row': cell.row,
              '--cell-column': cell.column,
            }}
          />
        ))}
      </div>
      <span className="mosaic-wallet-outline" />
      <span className="mosaic-wallet-brand">SyJ</span>
    </div>
  )
}
