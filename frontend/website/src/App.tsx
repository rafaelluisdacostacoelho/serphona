import React from 'react';
import { Routes, Route } from 'react-router-dom';
import Layout from '@components/Layout';
import HomePage from '@pages/HomePage';
import FeaturesPage from '@pages/FeaturesPage';
import PricingPage from '@pages/PricingPage';
import AboutPage from '@pages/AboutPage';
import ContactPage from '@pages/ContactPage';

function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="features" element={<FeaturesPage />} />
        <Route path="pricing" element={<PricingPage />} />
        <Route path="about" element={<AboutPage />} />
        <Route path="contact" element={<ContactPage />} />
      </Route>
    </Routes>
  );
}

export default App;
