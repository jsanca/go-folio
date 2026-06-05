import ProductList from './ProductList'
import { type Product } from '../components/ProductCard'

async function fetchProducts(): Promise<Product[]> {
  const gatewayURL = process.env.GATEWAY_URL ?? 'http://gateway-service:8090'
  const url = `${gatewayURL}/products`
  try {
    const res = await fetch(url, { cache: 'no-store' })
    if (!res.ok) {
      console.error('[products] fetch failed', { url, status: res.status })
      return []
    }
    const data = await res.json()
    console.log('[products] fetched', data.length, 'products')
    return data
  } catch (err) {
    console.error('[products] fetch error', { url, err })
    return []
  }
}

export const metadata = {
  title: 'Colección — S&J Heritage',
  description: 'Artículos de cuero artesanales S&J Heritage.',
}

export default async function ProductsPage() {
  const products = await fetchProducts()
  const active = products.filter((p) => p.active)
  const minioBase = process.env.NEXT_PUBLIC_MINIO_URL ?? ''

  return (
    <section className="bg-heritage-tan min-h-screen py-16 px-4">
      <div className="max-w-6xl mx-auto">
        <h1 className="font-serif text-center text-3xl font-bold tracking-[0.2em] text-heritage-dark mb-2 uppercase">
          Colección
        </h1>

        <ProductList products={active} minioBase={minioBase} />
      </div>
    </section>
  )
}
