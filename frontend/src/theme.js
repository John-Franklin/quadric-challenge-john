import { createTheme } from '@mui/material/styles'

// MUI theme built from the values in styles.txt.
const theme = createTheme({
  palette: {
    primary: { main: '#0066cc', dark: '#0052a3' },
    secondary: { main: '#e0e0e0', dark: '#d0d0d0', contrastText: '#333' },
    text: { primary: '#333', secondary: '#666' },
    background: { default: '#f5f5f5', paper: '#fff' },
    divider: '#e0e0e0',
  },
  typography: {
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif",
    fontSize: 14,
    h1: { fontSize: 24, fontWeight: 600, marginBottom: '1.5rem' },
  },
  shape: { borderRadius: 4 },
  components: {
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: {
        root: { textTransform: 'none', fontWeight: 600, padding: '10px 24px' },
      },
    },
    MuiFormHelperText: {
      styleOverrides: { root: { fontSize: 13, marginLeft: 0 } },
    },
    MuiTableCell: {
      styleOverrides: {
        root: { padding: '12px 16px', borderColor: '#e0e0e0' },
        head: { backgroundColor: '#f8f9fa', color: '#555', fontWeight: 600, borderBottomWidth: 2 },
      },
    },
  },
})

export default theme
