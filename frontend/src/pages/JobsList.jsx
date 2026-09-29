import { useState } from 'react'
import { Alert, Box, Button, Typography } from '@mui/material'
import CancelJobDialog from '../components/CancelJobDialog'
import InfiniteScrollTrigger from '../components/InfiniteScrollTrigger'
import JobDetailsDialog from '../components/JobDetailsDialog'
import JobsTable from '../components/JobsTable'
import LoadingIndicator from '../components/LoadingIndicator'
import PageHeading from '../components/PageHeading'
import useJobs from '../hooks/useJobs'
import { useAuth } from '../auth/AuthContext'

function JobsList() {
  const { user } = useAuth()
  const { jobs, total, loaded, hasMore, error, loadingMore, loadMoreError, loadMore, cancelJob } =
    useJobs()
  const [jobToCancel, setJobToCancel] = useState(null)
  const [jobForDetails, setJobForDetails] = useState(null)
  const [announcement, setAnnouncement] = useState('')

  return (
    <>
      <PageHeading>Jobs List</PageHeading>
      <span role="status" className="visually-hidden">{announcement}</span>
      <Box id="jobs-list">
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {!loaded ? (
          !error && <LoadingIndicator>Loading jobs...</LoadingIndicator>
        ) : jobs.length === 0 ? (
          <Typography color="text.secondary">No jobs yet.</Typography>
        ) : (
          <>
            <JobsTable jobs={jobs} onViewDetails={setJobForDetails} />

            {loadingMore && (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                <LoadingIndicator>Loading more jobs...</LoadingIndicator>
              </Box>
            )}

            {loadMoreError && (
              <Alert
                severity="error"
                sx={{ mt: 2 }}
                action={
                  <Button color="inherit" size="small" onClick={loadMore} sx={{ py: 0.25, px: 1 }}>
                    Retry
                  </Button>
                }
              >
                {loadMoreError}
              </Alert>
            )}

            {!hasMore && (
              <Typography color="text.secondary" align="center" sx={{ fontSize: 13, py: 2 }}>
                Showing all {total} jobs
              </Typography>
            )}

            {/* Paused after a failure so it doesn't retry in a loop; the Retry button resumes. */}
            {hasMore && (
              <InfiniteScrollTrigger onVisible={loadMore} disabled={loadingMore || Boolean(loadMoreError)} />
            )}
          </>
        )}
      </Box>
      <CancelJobDialog
        job={jobToCancel}
        onClose={() => setJobToCancel(null)}
        onConfirm={async (id) => {
          await cancelJob(id)
          setAnnouncement(`Job #${id} cancelled.`)
        }}
      />
      <JobDetailsDialog
        job={jobForDetails}
        onClose={() => setJobForDetails(null)}
        canCancel={user.is_admin || jobForDetails?.user_id === user.id}
        onCancel={(job) => {
          setJobForDetails(null)
          setJobToCancel(job)
        }}
      />
    </>
  )
}

export default JobsList
