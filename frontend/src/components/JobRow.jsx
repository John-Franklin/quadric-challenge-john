import { TableCell, TableRow } from '@mui/material'
import StatusBadge from './StatusBadge'
import { formatTimestamp } from '../utils/formatTimestamp'

function JobRow({ job }) {
  return (
    <TableRow hover>
      <TableCell>#{job.id}</TableCell>
      <TableCell>{job.action}</TableCell>
      <TableCell><StatusBadge status={job.status} /></TableCell>
      <TableCell sx={{ fontSize: 13, whiteSpace: 'nowrap' }}>
        <time dateTime={job.created_at}>{formatTimestamp(job.created_at)}</time>
      </TableCell>
      <TableCell>{job.notes}</TableCell>
    </TableRow>
  )
}

export default JobRow
