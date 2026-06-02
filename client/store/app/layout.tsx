import type { Metadata } from 'next'
import { Playfair_Display, Inter } from 'next/font/google'
import './globals.css'

const playfair = Playfair_Display({
  subsets: ['latin'],
  weight: ['400', '700', '900'],
  variable: '--font-playfair',
  display: 'swap',
})

const inter = Inter({
  subsets: ['latin'],
  weight: ['300', '400', '500'],
  variable: '--font-inter',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'S&J Heritage',
  description: 'Leather goods crafted for a lifetime.',
}

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${playfair.variable} ${inter.variable}`}>
      <body>
        <header className="bg-heritage-dark text-heritage-cream py-4">
          <div className="max-w-6xl mx-auto px-4 flex items-center justify-center">
            <span className="font-serif text-2xl font-bold tracking-widest">
              S&amp;J
            </span>
          </div>
          <nav />
        </header>

        <main>{children}</main>

        <footer className="bg-heritage-dark text-heritage-tan py-8 text-center text-sm">
          <p>
            &copy; {new Date().getFullYear()} S&amp;J Heritage. All rights
            reserved.
          </p>
        </footer>
      </body>
    </html>
  )
}
