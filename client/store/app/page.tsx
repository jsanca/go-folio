const FEATURED = [
  {
    title: 'Leather Bags',
    img: 'https://images.unsplash.com/photo-1548036328-c9fa89d128fa?w=800&q=80',
    alt: 'Leather handbag on a wooden surface',
  },
  {
    title: 'The Workshop',
    img: 'https://images.unsplash.com/photo-1473188588951-666fce8e7c68?w=800&q=80',
    alt: 'Artisan working leather in a workshop',
  },
  {
    title: 'Small Goods',
    img: 'https://images.unsplash.com/photo-1627123424574-724758594e93?w=800&q=80',
    alt: 'Leather wallets and small accessories',
  },
]

export default function Home() {
  return (
    <>
      {/* ── Hero ─────────────────────────────────────────────────────────── */}
      <section
        className="relative h-[650px] flex items-center"
        style={{
          backgroundImage:
            'url(https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=1600&q=80)',
          backgroundSize: 'cover',
          backgroundPosition: 'center',
        }}
      >
        {/* overlay */}
        <div className="absolute inset-0 bg-black/20" aria-hidden="true" />

        <div className="relative z-10 max-w-6xl mx-auto px-8 w-full">
          <p className="font-sans text-xs tracking-[0.3em] text-heritage-tan mb-4 uppercase">
            S&amp;J Heritage
          </p>
          <h1 className="font-serif text-5xl md:text-6xl font-black text-heritage-cream leading-tight mb-6 max-w-xl">
            TIMELESS
            <br />
            CRAFTSMANSHIP
          </h1>
          <p className="font-sans text-sm tracking-[0.25em] text-heritage-tan mb-10 uppercase">
            DISCOVER THE HERITAGE COLLECTION
          </p>
          <a
            href="/products"
            className="font-sans text-sm tracking-[0.3em] text-heritage-cream uppercase border-b border-heritage-cream pb-1 hover:text-heritage-tan hover:border-heritage-tan transition-colors"
          >
            SHOP NOW
          </a>
        </div>
      </section>

      {/* ── Featured Collections ─────────────────────────────────────────── */}
      <section className="bg-heritage-tan py-20 px-4">
        <h2 className="font-serif text-center text-3xl font-bold tracking-[0.2em] text-heritage-dark mb-12 uppercase">
          Featured Collections
        </h2>

        <div className="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-8">
          {FEATURED.map(({ title, img, alt }) => (
            <div key={title} className="group flex flex-col">
              <div className="overflow-hidden aspect-[4/5] mb-5">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={img}
                  alt={alt}
                  className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                />
              </div>
              <h3 className="font-serif text-lg font-bold text-heritage-dark tracking-widest uppercase mb-4">
                {title}
              </h3>
              <a
                href="/products"
                className="font-sans text-xs tracking-[0.25em] text-heritage-dark uppercase border-b border-heritage-dark pb-0.5 self-start hover:opacity-60 transition-opacity"
              >
                Explore
              </a>
            </div>
          ))}
        </div>
      </section>
    </>
  )
}
