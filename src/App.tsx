import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import JobTest from "./pages/JobTest";
import DashboardLayout from "./layouts/DashboardLayout";
import Dashboard from "./pages/Dashboard";
import Installation from "./pages/Installation";

function App() {
  return (
    <BrowserRouter>
      <DashboardLayout>
        <Routes>
          <Route path="/job-test" element={<JobTest />} />
          <Route path="/" element={<Dashboard />} />
          <Route path="/installation" element={<Installation />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </DashboardLayout>
    </BrowserRouter>
  );
}

export default App;
