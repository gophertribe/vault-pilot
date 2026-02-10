import { Link, useLocation } from "react-router-dom"
import {
  LayoutDashboard,
  Calendar,
  FolderOpen,
  CheckSquare,
  FolderKanban,
  Zap,
  MessageSquare,
  Settings,
} from "lucide-react"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

const navItems = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/calendar", label: "Calendar", icon: Calendar },
  { to: "/vault", label: "Vault", icon: FolderOpen },
  { to: "/tasks", label: "Tasks", icon: CheckSquare },
  { to: "/projects", label: "Projects", icon: FolderKanban },
  { to: "/automations", label: "Automations", icon: Zap },
  { to: "/chat", label: "Chat", icon: MessageSquare },
  { to: "/settings", label: "Settings", icon: Settings },
]

export function AppSidebar() {
  const location = useLocation()

  return (
    <Sidebar>
      <SidebarHeader className="border-b border-sidebar-border px-4 py-3">
        <span className="font-semibold text-sidebar-foreground">
          Life Pilot
        </span>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Navigation</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => {
                const Icon = item.icon
                const isActive = location.pathname === item.to
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive}
                    >
                      <Link
                        to={item.to}
                        className={cn(isActive && "bg-sidebar-accent")}
                      >
                        <Icon className="size-4" />
                        <span>{item.label}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
