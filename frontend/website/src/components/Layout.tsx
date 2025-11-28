import React from 'react';
import { Outlet, Link } from 'react-router-dom';

export default function Layout() {
  const CONSOLE_URL = 'http://localhost:3001';

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 sticky top-0 z-50">
        <nav className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo */}
            <Link to="/" className="flex items-center space-x-2">
              <span className="text-2xl">🐍</span>
              <span className="text-xl font-bold text-primary-600">Serphona</span>
            </Link>

            {/* Navigation */}
            <div className="hidden md:flex items-center space-x-8">
              <Link to="/features" className="text-gray-700 hover:text-primary-600 font-medium">
                Recursos
              </Link>
              <Link to="/pricing" className="text-gray-700 hover:text-primary-600 font-medium">
                Preços
              </Link>
              <Link to="/about" className="text-gray-700 hover:text-primary-600 font-medium">
                Sobre
              </Link>
              <Link to="/contact" className="text-gray-700 hover:text-primary-600 font-medium">
                Contato
              </Link>
            </div>

            {/* CTA Buttons */}
            <div className="flex items-center space-x-4">
              <a
                href={`${CONSOLE_URL}/login`}
                className="text-gray-700 hover:text-primary-600 font-medium"
              >
                Entrar
              </a>
              <a
                href={`${CONSOLE_URL}/register`}
                className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 font-medium transition-colors"
              >
                Começar Grátis
              </a>
            </div>
          </div>
        </nav>
      </header>

      {/* Main Content */}
      <main className="flex-1">
        <Outlet />
      </main>

      {/* Footer */}
      <footer className="bg-gray-900 text-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            {/* Company Info */}
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <span className="text-2xl">🐍</span>
                <span className="text-xl font-bold">Serphona</span>
              </div>
              <p className="text-gray-400 text-sm">
                Nuvem de agentes de IA inteligentes para atendimento automatizado.
              </p>
            </div>

            {/* Product */}
            <div>
              <h3 className="font-semibold mb-4">Produto</h3>
              <ul className="space-y-2 text-gray-400">
                <li><Link to="/features" className="hover:text-white">Recursos</Link></li>
                <li><Link to="/pricing" className="hover:text-white">Preços</Link></li>
                <li><a href={CONSOLE_URL} className="hover:text-white">Dashboard</a></li>
              </ul>
            </div>

            {/* Company */}
            <div>
              <h3 className="font-semibold mb-4">Empresa</h3>
              <ul className="space-y-2 text-gray-400">
                <li><Link to="/about" className="hover:text-white">Sobre</Link></li>
                <li><Link to="/contact" className="hover:text-white">Contato</Link></li>
              </ul>
            </div>

            {/* Legal */}
            <div>
              <h3 className="font-semibold mb-4">Legal</h3>
              <ul className="space-y-2 text-gray-400">
                <li><a href="#" className="hover:text-white">Privacidade</a></li>
                <li><a href="#" className="hover:text-white">Termos</a></li>
              </ul>
            </div>
          </div>

          <div className="mt-8 pt-8 border-t border-gray-800 text-center text-gray-400 text-sm">
            <p>&copy; 2024 Serphona. Todos os direitos reservados.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
