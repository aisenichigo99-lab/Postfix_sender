
import { useState } from "react";
import Sidebar from "../components/Sidebar";
import Header from "../components/Header";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="min-h-screen bg-[#10141d]">
      
      <Sidebar
        collapsed={collapsed}
        onToggle={() => setCollapsed(!collapsed)}
      />

      <div
        className={`transition-all duration-300 ${
          collapsed ? "ml-[76px]" : "ml-[250px]"
        }`}
      >
        <Header />

        <main className="p-8">
          {children}
        </main>
      </div>
    </div>
  );
}
