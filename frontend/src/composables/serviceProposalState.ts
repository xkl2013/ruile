export type ServiceFactProposalState = Record<string, unknown>

const resolvedServiceProposalStatuses = new Set(['confirmed', 'rejected'])

export function mergeServiceFactProposal(
  current: ServiceFactProposalState | undefined,
  incoming: ServiceFactProposalState,
): ServiceFactProposalState {
  if (!current) return incoming
  const currentStatus = String(current.status || '')
  const incomingStatus = String(incoming.status || '')
  if (
    resolvedServiceProposalStatuses.has(currentStatus)
    && !resolvedServiceProposalStatuses.has(incomingStatus)
  ) {
    return current
  }
  return { ...current, ...incoming }
}
