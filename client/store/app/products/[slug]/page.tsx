import { notFound } from 'next/navigation'
import VariantsDisplay, { type Variant } from '../../components/VariantsDisplay'

interface Product {
  id: number
  productCode: string
  title: string
  slug: string
  department: string
  category: string
  primaryImageUrl: string
  active: boolean
  variants: Variant[]
}

async function fetchProduct(slug: string): Promise<Product | null> {
  const gatewayURL = process.env.GATEWAY_URL ?? 'http://gateway-service:8090'
  try {
    const res = await fetch(`${gatewayURL}/products/${slug}`, {
      next: { revalidate: 60 },
    })
    if (res.status === 404) return null
    if (!res.ok) {
      console.error('[pdp] fetch failed', { slug, status: res.status })
      return null
    }
    return res.json()
  } catch (err) {
    console.error('[pdp] fetch error', { slug, err })
    return null
  }
}

export async function generateMetadata({
  params,
}: {
  params: { slug: string }
}) {
  const product = await fetchProduct(params.slug)
  if (!product) return { title: 'Producto no encontrado — S&J Heritage' }
  return {
    title: `${product.title} — S&J Heritage`,
    description: [product.department, product.category]
      .filter(Boolean)
      .join(', '),
  }
}

export default async function ProductPage({
  params,
}: {
  params: { slug: string }
}) {
  const product = await fetchProduct(params.slug)
  if (!product) notFound()

  const minioBase = process.env.NEXT_PUBLIC_MINIO_URL ?? ''
  const imageSrc =
    minioBase && product.primaryImageUrl
      ? `${minioBase}/${product.primaryImageUrl}`
      : null

  const label = [product.department, product.category]
    .filter(Boolean)
    .join(' · ')
  const activeVariants = product.variants.filter((v) => v.active)

  return (
    <section className="bg-heritage-tan min-h-screen py-12 px-4">
      <div className="max-w-6xl mx-auto">

        {/* Back link */}
        <a
          href="/products"
          className="font-sans text-xs tracking-[0.25em] uppercase text-heritage-dark/50 hover:text-heritage-dark transition-colors inline-flex items-center gap-2 mb-10"
        >
          <span aria-hidden>←</span> Colección
        </a>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-12 lg:gap-20">

          {/* ── Image ─────────────────────────────────────────────────── */}
          <div className="aspect-[4/5] bg-heritage-dark/10 overflow-hidden">
            {imageSrc ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={imageSrc}
                alt={product.title}
                className="w-full h-full object-cover"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center">
                <span className="font-sans text-xs tracking-widest uppercase text-heritage-dark/30">
                  S&amp;J
                </span>
              </div>
            )}
          </div>

          {/* ── Details ───────────────────────────────────────────────── */}
          <div className="flex flex-col justify-start pt-2">

            {label && (
              <p className="font-sans text-xs tracking-[0.25em] uppercase text-heritage-dark/50 mb-3">
                {label}
              </p>
            )}

            <h1 className="font-serif text-3xl font-bold text-heritage-dark tracking-wide leading-tight mb-2">
              {product.title}
            </h1>

            <p className="font-sans text-xs tracking-[0.2em] text-heritage-dark/40 uppercase mb-10">
              {product.productCode}
            </p>

            {/* Live stock — client component subscribes to SSE */}
            <VariantsDisplay variants={activeVariants} />

          </div>

        </div>
      </div>
    </section>
  )
}
