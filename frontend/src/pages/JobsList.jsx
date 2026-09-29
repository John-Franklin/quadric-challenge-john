import { Alert, Box, CircularProgress, Typography } from '@mui/material'
import JobsTable from '../components/JobsTable'
import useJobs from '../hooks/useJobs'

function JobsList() {
  const { jobs, loading, error } = useJobs()

  return (
    <>
      <Typography variant="h1">Jobs List</Typography>
      <Box id="jobs-list">
        {loading ? (
          <Box role="status" sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <CircularProgress size={20} />
            <Typography color="text.secondary">Loading jobs...</Typography>
          </Box>
        ) : error ? (
          <Alert severity="error">{error}</Alert>
        ) : jobs.length === 0 ? (
          <Typography color="text.secondary">No jobs yet.</Typography>
        ) : (
          <JobsTable jobs={jobs} />
        )}
      </Box>
    </>
  )
}

export default JobsList
