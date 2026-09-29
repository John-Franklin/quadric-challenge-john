import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from '@mui/material'
import JobRow from './JobRow'

function JobsTable({ jobs }) {
  return (
    <TableContainer>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Action</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Created</TableCell>
            <TableCell>Notes</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {jobs.map((job) => (
            <JobRow key={job.id} job={job} />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

export default JobsTable
