import { Button, TableCell, TableRow } from '@mui/material'
import StatusBadge from './StatusBadge'
import { formatTimestamp } from '../utils/formatTimestamp'

function JobRow({ job, onViewDetails }) {
  return (
    <TableRow hover>
      <TableCell component="th" scope="row">#{job.id}</TableCell>
      <TableCell>{job.owner_email || '—'}</TableCell>
      <TableCell>{job.action}</TableCell>
      <TableCell><StatusBadge status={job.status} /></TableCell>
      <TableCell sx={{ fontSize: 13, whiteSpace: 'nowrap' }}>
        <time dateTime={job.created_at}>{formatTimestamp(job.created_at)}</time>
      </TableCell>
      <TableCell>{job.notes}</TableCell>
      <TableCell align="right">
        <Button
          size="small"
          variant="outlined"
          onClick={() => onViewDetails(job)}
          aria-label={`View details for job #${job.id}`}
          sx={{ py: 0.25, px: 1.5, whiteSpace: 'nowrap' }}
        >
          Details
        </Button>
      </TableCell>
    </TableRow>
  )
}

export default JobRow
