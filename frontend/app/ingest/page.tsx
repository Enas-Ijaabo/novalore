import { getIngestStatus } from '../lib/api'
import IngestTab from '../components/IngestTab'

export const dynamic = 'force-dynamic'

export default async function IngestPage() {
  const data = await getIngestStatus()
  return <IngestTab initialData={data} />
}
