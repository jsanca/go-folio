'use client'

import { useState } from 'react'
import ProductCard, { type Product } from '../components/ProductCard'

const TABS = ['Todos', 'Ellas', 'Ellos', 'Calzado', 'Accesorios'] as const
type Tab = (typeof TABS)[number]

interface Props {
  products: Product[]
  minioBase: string
}

export default function ProductList({ products, minioBase }: Props) {
  const [tab, setTab] = useState<Tab>('Todos')

  const filtered =
    tab === 'Todos' ? products : products.filter((p) => p.department === tab)

  return (
    <>
      {/* ── Filter tabs ─────────────────────────────────────────────── */}
      <div className="flex flex-wrap gap-2 justify-center mb-10">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={[
              'font-sans text-xs tracking-[0.2em] uppercase px-4 py-2 border transition-colors',
              tab === t
                ? 'bg-heritage-dark text-heritage-cream border-heritage-dark'
                : 'bg-transparent text-heritage-dark/60 border-heritage-dark/20 hover:border-heritage-dark hover:text-heritage-dark',
            ].join(' ')}
          >
            {t}
          </button>
        ))}
      </div>

      {/* ── Count ───────────────────────────────────────────────────── */}
      <p className="font-sans text-center text-xs tracking-[0.3em] text-heritage-dark/50 uppercase mb-12">
        {filtered.length > 0 ? `${filtered.length} productos` : ''}
      </p>

      {/* ── Grid ────────────────────────────────────────────────────── */}
      {filtered.length === 0 ? (
        <p className="font-sans text-sm text-heritage-dark/60 text-center py-24">
          Catálogo no disponible en este momento.
        </p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-8">
          {filtered.map((product) => (
            <ProductCard key={product.id} product={product} minioBase={minioBase} />
          ))}
        </div>
      )}
    </>
  )
}
