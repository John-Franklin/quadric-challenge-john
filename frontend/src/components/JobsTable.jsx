import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from '@mui/material'
import JobRow from './JobRow'

function JobsTable({ jobs, onViewDetails }) {
  return (
    <TableContainer>
      <Table aria-label="Jobs">
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Owner</TableCell>
            <TableCell>Action</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Created</TableCell>
            <TableCell>Notes</TableCell>
            <TableCell align="right">
              <span className="visually-hidden">Actions</span>
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {jobs.map((job) => (
            <JobRow key={job.id} job={job} onViewDetails={onViewDetails} />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default JobsTable
