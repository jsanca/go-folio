'use client'

import { useEffect, useState } from 'react'

export interface Stock {
  available: number
  reserved: number
  stockStatus: 'IN_STOCK' | 'LOW_STOCK' | 'OUT_OF_STOCK'
}

export interface Variant {
  sku: string
  colorName: string
  colorSlug: string
  primaryColorHex: string
  retailPrice: { amountCents: number }
  currency: string
  stock: Stock
  active: boolean
}

interface SSEEvent {
  eventType: string
  sku: string
  available: number
  reserved: number
  status: Stock['stockStatus']
}

interface Props {
  variants: Variant[]
}

function formatPrice(amountCents: number, currency: string): string {
  return `${(amountCents / 100).toLocaleString('es-CR')} ${currency}`
}

function StockBadge({ status }: { status: Stock['stockStatus'] }) {
  if (status === 'IN_STOCK') return null
  if (status === 'LOW_STOCK')
    return (
      <span className="font-sans text-[10px] tracking-[0.15em] uppercase text-amber-700">
        Últimas unidades
      </span>
    )
  return (
    <span className="font-sans text-[10px] tracking-[0.15em] uppercase text-heritage-dark/40">
      Agotado
    </span>
  )
}

export default function VariantsDisplay({ variants: initialVariants }: Props) {
  const [variants, setVariants] = useState<Variant[]>(initialVariants)

  useEffect(() => {
    const skus = new Set(initialVariants.map((v) => v.sku))
    const gatewayURL =
      process.env.NEXT_PUBLIC_GATEWAY_URL ?? 'http://localhost:8090'
    const es = new EventSource(`${gatewayURL}/events`)

    es.onmessage = (e: MessageEvent) => {
      try {
        const evt: SSEEvent = JSON.parse(e.data)
        if (!skus.has(evt.sku)) return
        setVariants((prev) =>
          prev.map((v) =>
            v.sku === evt.sku
              ? {
                  ...v,
                  stock: {
                    available: evt.available,
                    reserved: evt.reserved,
                    stockStatus: evt.status,
                  },
                }
              : v,
          ),
        )
      } catch {
        // malformed event — ignore
      }
    }

    return () => es.close()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const activeVariants = variants.filter((v) => v.active)

  if (activeVariants.length === 0) {
    return (
      <p className="font-sans text-sm text-heritage-dark/50 italic">
        Consultar disponibilidad
      </p>
    )
  }

  if (activeVariants.length === 1 && !activeVariants[0].colorName) {
    const v = activeVariants[0]
    return (
      <div>
        {v.retailPrice.amountCents > 0 ? (
          <p className="font-sans text-lg text-heritage-dark">
            {formatPrice(v.retailPrice.amountCents, v.currency)}
          </p>
        ) : (
          <p className="font-sans text-sm text-heritage-dark/60 italic">
            Consultar precio
          </p>
        )}
        <StockBadge status={v.stock.stockStatus} />
      </div>
    )
  }

  return (
    <div>
      <p className="font-sans text-xs tracking-[0.2em] uppercase text-heritage-dark/50 mb-4">
        Variantes
      </p>
      <div className="divide-y divide-heritage-dark/10">
        {activeVariants.map((v) => {
          const dimmed = v.stock.stockStatus === 'OUT_OF_STOCK'
          return (
            <div
              key={v.sku}
              className={`flex items-center gap-4 py-3 transition-opacity ${dimmed ? 'opacity-40' : ''}`}
            >
              {v.primaryColorHex && (
                <span
                  className="w-4 h-4 rounded-full border border-heritage-dark/20 flex-shrink-0"
                  style={{ backgroundColor: v.primaryColorHex }}
                  title={v.colorName}
                />
              )}

              <span className="font-sans text-sm text-heritage-dark flex-1 leading-none">
                {v.colorName || v.sku}
              </span>

              <span className="font-sans text-sm text-heritage-dark text-right">
                {v.retailPrice.amountCents > 0 ? (
                  formatPrice(v.retailPrice.amountCents, v.currency)
                ) : (
                  <span className="italic text-heritage-dark/50">
                    Consultar precio
                  </span>
                )}
              </span>

              <div className="w-28 text-right">
                <StockBadge status={v.stock.stockStatus} />
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
