import { Link, Route, Routes } from 'react-router'
import { LoginPage } from './auth/LoginPage'
import { RequireAuth } from './auth/RequireAuth'
import { InventoryPage } from './inventory/InventoryPage'
import { Layout } from './Layout'
import { VehiclePage } from './vehicle/VehiclePage'

function NotFound() {
  return (
    <main className="content">
      <h1>Page not found</h1>
      <p>
        <Link to="/">Go to the inventory</Link>
      </p>
    </main>
  )
}

/** The route table. Providers and the router wrap it in main.tsx (and in tests). */
export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route element={<Layout />}>
          <Route index element={<InventoryPage />} />
          <Route path="vehicles/:vehicleId" element={<VehiclePage />} />
        </Route>
      </Route>
      <Route path="*" element={<NotFound />} />
    </Routes>
  )
}
