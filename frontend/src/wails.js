// wails.js — thin wrapper around window.go.main.App.*
// Provides typed helpers and centralises error handling.

const App = () => window.go.main.App

export const ListProjects = () => App().ListProjects()
export const AddProject = (path, name, targetOverride) => App().AddProject(path, name, targetOverride ?? null)
export const RemoveProject = id => App().RemoveProject(id)
export const ListWSLDistros = () => App().ListWSLDistros()

export const AddServer = (projectID, srv) => App().AddServer(projectID, srv)
export const UpdateServer = (projectID, srv) => App().UpdateServer(projectID, srv)
export const RemoveServer = (projectID, serverID) => App().RemoveServer(projectID, serverID)

export const StartServer = id => App().StartServer(id)
export const StopServer = id => App().StopServer(id)
export const RestartServer = id => App().RestartServer(id)
export const ResyncServerPort = id => App().ResyncServerPort(id)

export const ScanSystemPorts = () => App().ScanSystemPorts()
export const KillByPort = port => App().KillByPort(port)
export const SuggestFreePort = from => App().SuggestFreePort(from)

export const GetRecentLogs = (sid, n) => App().GetRecentLogs(sid, n)
export const GetSystemStats = () => App().GetSystemStats()
export const ExportLogs = (sid, dst) => App().ExportLogs(sid, dst)

export const AnalyzeProject    = path   => App().AnalyzeProject(path)
export const GetListeningPorts = ()     => App().GetListeningPorts()
export const BrowseDirectory   = ()     => App().BrowseDirectory()
export const SetAutostart        = enable => App().SetAutostart(enable)
export const GetAllServerStates  = ()     => App().GetAllServerStates()

// Wails event subscription
export const onEvent = (name, cb) => window.runtime.EventsOn(name, cb)
export const offEvent = name => window.runtime.EventsOff(name)

// Open URL in the OS default browser (not a WebView2 popup)
export const BrowserOpenURL = url => window.runtime.BrowserOpenURL(url)

// Window controls (frameless custom titlebar + mini mode)
export const WindowMinimise       = ()     => window.runtime.WindowMinimise()
export const WindowToggleMaximise = ()     => window.runtime.WindowToggleMaximise()
export const QuitApp              = ()     => window.runtime.Quit()
export const WindowGetSize        = ()     => window.runtime.WindowGetSize()
export const WindowSetSize        = (w, h) => window.runtime.WindowSetSize(w, h)
export const WindowSetAlwaysOnTop = b      => window.runtime.WindowSetAlwaysOnTop(b)
