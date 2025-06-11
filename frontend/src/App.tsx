import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import ServiceSources from './pages/ServiceSources'
import ServiceList from './pages/ServiceList'
import RouteConfig from './pages/RouteConfig'
import DomainManagement from './pages/DomainManagement'
import CertificateManagement from './pages/CertificateManagement'
import LocalModelManagement from './pages/LocalModelManagement'
import './App.css'

/**
 * Main application component that sets up the routing structure.
 * Defines all the route paths and their corresponding components.
 * Wrapped with Layout component for consistent UI structure.
 */
function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/service-sources" element={<ServiceSources />} />
          <Route path="/service-list" element={<ServiceList />} />
          <Route path="/route-config" element={<RouteConfig />} />
          <Route path="/domain-management" element={<DomainManagement />} />
          <Route path="/certificate-management" element={<CertificateManagement />} />
          <Route path="/local-models" element={<LocalModelManagement />} />
        </Routes>
      </Layout>
    </Router>
  )
}

export default App
