import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import Admin from './Admin.tsx'

// Simple router based on pathname
const isAdminPage = window.location.pathname === '/admin' || window.location.pathname === '/admin/';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {isAdminPage ? <Admin /> : <App />}
  </StrictMode>,
)
