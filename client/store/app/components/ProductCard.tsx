export interface ProductVariant {
  sku: string
  retailPrice: { amountCents: number }
  currency: string
  active: boolean
}

export interface Product {
  id: number
  productCode: string
  title: string
  slug: string
  department: string
  category: string
  primaryImageUrl: string
  active: boolean
  variants: ProductVariant[]
}

interface Props {
  product: Product
  minioBase: string
}

export default function ProductCard({ product, minioBase }: Props) {
  const src =
    minioBase && product.primaryImageUrl
      ? `${minioBase}/${product.primaryImageUrl}`
      : null

  const firstVariant = product.variants?.[0]
  const label = [product.department, product.category].filter(Boolean).join(' · ')

  return (
    <div className="group flex flex-col">
      <div className="overflow-hidden aspect-[4/5] mb-4 bg-heritage-dark/10">
        {src ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={src}
            alt={product.title}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
          />
        ) : (
          <div className="w-full h-full bg-heritage-tan flex items-center justify-center">
            <span className="font-sans text-xs tracking-widest uppercase text-heritage-dark/30">
              S&amp;J
            </span>
          </div>
        )}
      </div>

      {label && (
        <p className="font-sans text-xs tracking-[0.2em] text-heritage-dark/50 uppercase mb-1">
          {label}
        </p>
      )}

      <h3 className="font-serif text-sm font-bold text-heritage-dark tracking-wide leading-snug mb-3">
        {product.title}
      </h3>

      {firstVariant && firstVariant.retailPrice.amountCents > 0 && (
        <p className="font-sans text-sm text-heritage-dark mb-3">
          {(firstVariant.retailPrice.amountCents / 100).toLocaleString('es-CR')}{' '}
          {firstVariant.currency}
        </p>
      )}

      <a
        href={`/products/${product.slug}`}
        className="font-sans text-xs tracking-[0.25em] text-heritage-dark uppercase border-b border-heritage-dark pb-0.5 self-start hover:opacity-60 transition-opacity mt-auto"
      >
        Ver más
      </a>
    </div>
  )
}
