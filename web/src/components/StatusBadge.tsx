import type { Finding } from '../api'

const LABELS: Record<Finding['status'], string> = {
  new: 'New',
  acknowledged: 'Acknowledged',
  resolved: 'Resolved',
  false_positive: 'False positive',
}

export default function StatusBadge({ status }: { status: Finding['status'] }) {
  return <span className={`status-badge status-${status}`}>{LABELS[status]}</span>
}
