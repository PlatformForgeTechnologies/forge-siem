import React, { useState, useEffect } from 'react';
import { Box, Typography, Container, Button, createTheme, ThemeProvider, CssBaseline, Paper, Grid, CircularProgress } from '@mui/material';
import { Routes, Route, Outlet, Link as RouterLink } from 'react-router-dom';

interface Agent {
  id: string;
  hostname: string;
  group: string;
  status: 'active' | 'inactive' | 'error';
}

interface Alert {
  id: string;
  title: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  mitre: string;
}

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    background: {
      default: '#08111a',
      paper: 'rgba(12, 27, 40, 0.88)',
    },
    primary: {
      main: '#70f0b4', // Accent color
    },
    secondary: {
      main: '#ff8a5b', // Warn color
    },
    error: {
      main: '#ff4f4f', // Critical color
    },
    text: {
      primary: '#e7f0f6',
      secondary: '#8ea8ba',
    },
  },
  typography: {
    fontFamily: '"IBM Plex Sans", "Segoe UI", sans-serif',
  },
  components: {
    MuiPaper: {
      styleOverrides: {
        root: {
          border: '1px solid rgba(130, 175, 202, 0.22)',
          borderRadius: '24px',
          boxShadow: '0 24px 80px rgba(0, 0, 0, 0.35)',
          backdropFilter: 'blur(12px)',
          background: 'rgba(12, 27, 40, 0.88)', // Ensure panel background from CSS variables is used
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: '999px',
          border: '1px solid rgba(255, 255, 255, 0.12)',
          background: 'rgba(255, 255, 255, 0.04)',
          color: '#e7f0f6', // Ensure text color from CSS variables is used
          textTransform: 'none', // Prevent uppercase transformation by default
          '&:hover': {
            background: 'rgba(255, 255, 255, 0.08)',
          },
        },
      },
    },
  },
});

function DashboardLayout() {
  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />
      <Container component="main" maxWidth="xl" sx={{ width: 'min(1160px, calc(100% - 32px))', margin: '0 auto', paddingY: '40px' }}>
        {/* Navigation or Header could go here */}
        <Outlet /> {/* This is where nested routes will render */}
      </Container>
    </ThemeProvider>
  );
}

function DashboardHome() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loadingAgents, setLoadingAgents] = useState(true);
  const [errorAgents, setErrorAgents] = useState<string | null>(null);
  const [loadingAlerts, setLoadingAlerts] = useState(true);
  const [errorAlerts, setErrorAlerts] = useState<string | null>(null);

  useEffect(() => {
    const fetchAgents = async () => {
      try {
        const response = await fetch('/api/agents');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data: Agent[] = await response.json();
        // Simulate data for now if API is not yet available
        setAgents(data.length > 0 ? data : [
            { id: "agent-1", hostname: "ip-10-0-0-10", group: "eks", status: "active" },
            { id: "agent-2", hostname: "ip-10-0-2-23", group: "ecs", status: "active" }
        ]);
      } catch (error: any) {
        setErrorAgents(error.message);
        // Fallback to simulated data on error
        setAgents([
            { id: "agent-1", hostname: "ip-10-0-0-10", group: "eks", status: "active" },
            { id: "agent-2", hostname: "ip-10-0-2-23", group: "ecs", status: "active" }
        ]);
      } finally {
        setLoadingAgents(false);
      }
    };

    const fetchAlerts = async () => {
      try {
        const response = await fetch('/api/alerts');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data: Alert[] = await response.json();
        // Simulate data for now if API is not yet available
        setAlerts(data.length > 0 ? data : [
            { id: "alert-1", title: "SSH brute force", severity: "high", mitre: "T1110" },
            { id: "alert-2", title: "/etc/shadow modified", severity: "critical", mitre: "T1098" }
        ]);
      } catch (error: any) {
        setErrorAlerts(error.message);
        // Fallback to simulated data on error
        setAlerts([
            { id: "alert-1", title: "SSH brute force", severity: "high", mitre: "T1110" },
            { id: "alert-2", hostname: "ip-10-0-2-23", group: "ecs", status: "active" }
        ]);
      } finally {
        setLoadingAlerts(false);
      }
    };

    fetchAgents();
    fetchAlerts();
  }, []);

  return (
    <>
        <Box
          sx={{
            display: 'grid',
            gap: '20px',
            gridTemplateColumns: { xs: '1fr', md: '1.3fr 1fr' }, // Responsive columns
            alignItems: 'end',
            marginBottom: '20px',
            paddingTop: '20px', // Add some top padding
          }}
        >
          <Box>
            <Typography
              variant="overline"
              sx={{
                color: 'primary.main',
                letterSpacing: '0.2em',
                textTransform: 'uppercase',
                fontSize: '12px',
                display: 'block',
              }}
            >
              Internal SOC
            </Typography>
            <Typography
              variant="h1"
              sx={{
                margin: 0,
                fontSize: { xs: '42px', sm: 'clamp(42px, 7vw, 76px)' },
                lineHeight: 0.95,
                color: 'text.primary',
              }}
            >
              Forge SIEM
            </Typography>
            <Typography
              variant="body1"
              sx={{
                maxWidth: '54ch',
                color: 'text.secondary',
                marginTop: '10px',
              }}
            >
              Single-tenant detection, endpoint telemetry, and active response for company infrastructure.
            </Typography>
          </Box>
          <Box
            sx={{
              background: 'background.paper',
              border: '1px solid',
              borderColor: 'rgba(130, 175, 202, 0.22)',
              borderRadius: '24px',
              boxShadow: '0 24px 80px rgba(0, 0, 0, 0.35)',
              backdropFilter: 'blur(12px)',
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, 1fr)' }, // Responsive stats grid
              padding: '22px',
            }}
          >
            <Box>
              <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block' }}>
                Agents
              </Typography>
              <Typography variant="h4" component="strong" sx={{ display: 'block', fontSize: '36px', marginTop: '6px' }}>
                {loadingAgents ? <CircularProgress size={24} /> : agents.length}
              </Typography>
            </Box>
            <Box>
              <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block' }}>
                Open alerts
              </Typography>
              <Typography variant="h4" component="strong" sx={{ display: 'block', fontSize: '36px', marginTop: '6px' }}>
                {loadingAlerts ? <CircularProgress size={24} /> : alerts.length}
              </Typography>
            </Box>
            <Box>
              <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block' }}>
                Events / 24h
              </Typography>
              <Typography variant="h4" component="strong" sx={{ display: 'block', fontSize: '36px', marginTop: '6px' }}>
                92.8k
              </Typography>
            </Box>
          </Box>
        </Box>

        <Grid container spacing={2} sx={{ marginTop: '20px' }}>
          <Grid item xs={12} md={6}>
            <Paper sx={{ padding: '22px', height: '100%' }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <Typography variant="h5" component="h2">
                  Agents
                </Typography>
                <Button component={RouterLink} to="/opensearch-dashboards" target="_blank" rel="noopener noreferrer" sx={{ textTransform: 'none' }}>
                  OpenSearch Dashboards
                </Button>
              </Box>
              {loadingAgents ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', paddingY: '20px' }}>
                  <CircularProgress />
                </Box>
              ) : errorAgents ? (
                <Typography color="error">{errorAgents}</Typography>
              ) : (
                <Box component="ul" sx={{ listStyle: 'none', padding: 0, margin: 0 }}>
                  {agents.map((agent) => (
                    <Box
                      component="li"
                      key={agent.id}
                      sx={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        paddingY: '14px',
                        borderTop: '1px solid rgba(142, 168, 186, 0.16)',
                        '&:first-of-type': { borderTop: 'none' },
                      }}
                    >
                      <Box>
                        <Typography variant="subtitle1" component="strong">
                          {agent.hostname}
                        </Typography>
                        <Typography variant="caption" sx={{ color: 'text.secondary', marginLeft: '8px' }}>
                          {agent.group}
                        </Typography>
                      </Box>
                      <Typography
                        variant="caption"
                        sx={{
                          padding: '6px 10px',
                          borderRadius: '999px',
                          fontSize: '12px',
                          textTransform: 'uppercase',
                          background: agent.status === 'active' ? 'rgba(112, 240, 180, 0.14)' : 'transparent',
                          color: agent.status === 'active' ? 'primary.main' : 'text.primary',
                        }}
                      >
                        {agent.status}
                      </Typography>
                    </Box>
                  ))}
                </Box>
              )}
            </Paper>
          </Grid>

          <Grid item xs={12} md={6}>
            <Paper sx={{ padding: '22px', height: '100%' }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <Typography variant="h5" component="h2">
                  Alerts
                </Typography>
                <Button>Stream live</Button>
              </Box>
              {loadingAlerts ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', paddingY: '20px' }}>
                  <CircularProgress />
                </Box>
              ) : errorAlerts ? (
                <Typography color="error">{errorAlerts}</Typography>
              ) : (
                <Box component="ul" sx={{ listStyle: 'none', padding: 0, margin: 0 }}>
                  {alerts.map((alert) => (
                    <Box
                      component="li"
                      key={alert.id}
                      sx={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        paddingY: '14px',
                        borderTop: '1px solid rgba(142, 168, 186, 0.16)',
                        '&:first-of-type': { borderTop: 'none' },
                      }}
                    >
                      <Box>
                        <Typography variant="subtitle1" component="strong">
                          {alert.title}
                        </Typography>
                        <Typography variant="caption" sx={{ color: 'text.secondary', marginLeft: '8px' }}>
                          {alert.mitre}
                        </Typography>
                      </Box>
                      <Typography
                        variant="caption"
                        sx={{
                          padding: '6px 10px',
                          borderRadius: '999px',
                          fontSize: '12px',
                          textTransform: 'uppercase',
                          background: alert.severity === 'high' ? 'rgba(255, 138, 91, 0.14)' : alert.severity === 'critical' ? 'rgba(255, 79, 79, 0.14)' : 'transparent',
                          color: alert.severity === 'high' ? 'secondary.main' : alert.severity === 'critical' ? 'error.main' : 'text.primary',
                        }}
                      >
                        {alert.severity}
                      </Typography>
                    </Box>
                  ))}
                </Box>
              )}
            </Paper>
          </Grid>
        </Grid>
    </>
  );
}

export function App() {
  return (
    <Routes>
      <Route path="/" element={<DashboardLayout />}>
        <Route index element={<DashboardHome />} />
        {/* Placeholder for Agents List page */}
        <Route path="agents" element={<Typography variant="h4" sx={{ color: 'text.primary' }}>Agents List Page</Typography>} />
        {/* Placeholder for Alerts List page */}
        <Route path="alerts" element={<Typography variant="h4" sx={{ color: 'text.primary' }}>Alerts List Page</Typography>} />
        {/* Fallback for unknown routes */}
        <Route path="*" element={<Typography variant="h4" sx={{ color: 'text.primary' }}>404 Not Found</Typography>} />
      </Route>
    </Routes>
  );
}
