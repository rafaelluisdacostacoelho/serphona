import React from 'react';
import { Link } from 'react-router-dom';

export default function HomePage() {
  const CONSOLE_URL = 'http://localhost:3001';

  return (
    <div>
      {/* Hero Section */}
      <section className="bg-gradient-to-br from-primary-50 to-white py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h1 className="text-5xl md:text-6xl font-extrabold text-gray-900 mb-6">
              Nuvem de Agentes de IA
              <span className="block text-primary-600">para Atendimento Inteligente</span>
            </h1>
            <p className="text-xl text-gray-600 mb-8 max-w-3xl mx-auto">
              Automatize seu atendimento com agentes de IA contextualizados para o seu negócio.
              Selfservice completo para criar e gerenciar sua própria nuvem de agentes.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <a
                href={`${CONSOLE_URL}/register`}
                className="px-8 py-4 bg-primary-600 text-white rounded-lg hover:bg-primary-700 font-semibold text-lg transition-colors"
              >
                Começar Grátis
              </a>
              <Link
                to="/features"
                className="px-8 py-4 bg-white text-primary-600 border-2 border-primary-600 rounded-lg hover:bg-primary-50 font-semibold text-lg transition-colors"
              >
                Ver Recursos
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold text-gray-900 mb-4">
              Por que escolher Serphona?
            </h2>
            <p className="text-xl text-gray-600">
              Tecnologia de ponta para automatizar seu atendimento
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <div className="p-6 bg-gray-50 rounded-xl">
              <div className="text-4xl mb-4">🤖</div>
              <h3 className="text-xl font-semibold mb-2">Agentes Inteligentes</h3>
              <p className="text-gray-600">
                Crie agentes personalizados com contexto do seu negócio
              </p>
            </div>

            <div className="p-6 bg-gray-50 rounded-xl">
              <div className="text-4xl mb-4">⚡</div>
              <h3 className="text-xl font-semibold mb-2">Selfservice Completo</h3>
              <p className="text-gray-600">
                Gerencie tudo através do nosso dashboard intuitivo
              </p>
            </div>

            <div className="p-6 bg-gray-50 rounded-xl">
              <div className="text-4xl mb-4">📊</div>
              <h3 className="text-xl font-semibold mb-2">Analytics Avançado</h3>
              <p className="text-gray-600">
                Acompanhe métricas e melhore continuamente
              </p>
            </div>

            <div className="p-6 bg-gray-50 rounded-xl">
              <div className="text-4xl mb-4">🔧</div>
              <h3 className="text-xl font-semibold mb-2">Integrações</h3>
              <p className="text-gray-600">
                Conecte com WhatsApp, Telegram, e-mail e mais
              </p>
            </div>

            <div className="p-6 bg-gray-50 rounded-xl">
              <div className="text-4xl mb-4">🔒</div>
              <h3 className="text-xl font-semibold mb-2">Seguro</h3>
              <p className="text-gray-600">
                Multi-tenant com isolamento completo de dados
              </p>
            </div>

            <div className="p-6 bg-gray-50 rounded-xl">
              <div className="text-4xl mb-4">🚀</div>
              <h3 className="text-xl font-semibold mb-2">Escalável</h3>
              <p className="text-gray-600">
                Infraestrutura preparada para crescer com você
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 bg-primary-600">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-4xl font-bold text-white mb-4">
            Pronto para começar?
          </h2>
          <p className="text-xl text-primary-100 mb-8">
            Crie sua conta gratuitamente e configure seus primeiros agentes em minutos
          </p>
          <a
            href={`${CONSOLE_URL}/register`}
            className="inline-block px-8 py-4 bg-white text-primary-600 rounded-lg hover:bg-gray-100 font-semibold text-lg transition-colors"
          >
            Começar Agora
          </a>
        </div>
      </section>
    </div>
  );
}
