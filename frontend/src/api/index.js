import { createHttpClient } from './httpClient'

// The app talks to the backend only through this object. To swap transports (mock, GraphQL, etc.),
// provide another implementation with the same `auth`, `jobs`, and `admin` methods, each returning
// a promise that rejects with an Error (and `status` when available) and honouring `signal` aborts.
const api = createHttpClient()

export default api
