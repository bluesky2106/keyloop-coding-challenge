import { useId, useState, type SyntheticEvent } from 'react'
import { useLogAction } from '../api/hooks'
import { ApiError } from '../api/errors'
import type { ActionType } from '../api/types'
import { ACTION_LABELS, ACTION_TYPES } from '../shared/labels'

const MAX_NOTE = 500

function describe(error: Error): string {
  if (error instanceof ApiError && error.status === 409) {
    return 'This vehicle is not aging, so it cannot take an action.'
  }
  return error instanceof ApiError && error.code === 'validation_error'
    ? error.message
    : 'Could not save the action. Please try again.'
}

export function ActionForm({ vehicleId }: { vehicleId: string }) {
  const log = useLogAction(vehicleId)
  const [actionType, setActionType] = useState<ActionType | ''>('')
  const [note, setNote] = useState('')
  const fields = log.error instanceof ApiError ? log.error.fields : {}
  const id = useId()
  const clearOutcome = () => {
    if (!log.isIdle) log.reset()
  }
  const typeErrorId = `${id}-type-error`
  const noteHintId = `${id}-note-hint`
  const noteErrorId = `${id}-note-error`

  function onSubmit(event: SyntheticEvent): void {
    event.preventDefault()
    if (actionType === '') return
    log.mutate(
      { action_type: actionType, note: note.trim() === '' ? null : note },
      {
        onSuccess: () => {
          setActionType('')
          setNote('')
        },
      },
    )
  }

  return (
    <form onSubmit={onSubmit} className="card action-form">
      <h2>Log an action</h2>

      <div className="field">
        <label htmlFor={`${id}-type`}>Action</label>
        <select
          id={`${id}-type`}
          value={actionType}
          onChange={(e) => {
            setActionType(e.target.value as ActionType | '')
            clearOutcome()
          }}
          required
          aria-invalid={Boolean(fields.action_type)}
          aria-describedby={fields.action_type ? typeErrorId : undefined}
        >
          <option value="">Choose an action…</option>
          {ACTION_TYPES.map((type) => (
            <option key={type} value={type}>
              {ACTION_LABELS[type]}
            </option>
          ))}
        </select>
        {fields.action_type && (
          <span id={typeErrorId} className="error">
            {fields.action_type}
          </span>
        )}
      </div>

      <div className="field">
        <label htmlFor={`${id}-note`}>Note (optional)</label>
        <textarea
          id={`${id}-note`}
          value={note}
          onChange={(e) => {
            setNote(e.target.value)
            clearOutcome()
          }}
          maxLength={MAX_NOTE}
          rows={3}
          aria-invalid={Boolean(fields.note)}
          aria-describedby={fields.note ? `${noteHintId} ${noteErrorId}` : noteHintId}
        />
        <span id={noteHintId} className="muted">
          {note.length} / {MAX_NOTE}
        </span>
        {fields.note && (
          <span id={noteErrorId} className="error">
            {fields.note}
          </span>
        )}
      </div>

      {log.isError && (
        <p role="alert" className="error">
          {describe(log.error)}
        </p>
      )}
      {log.isSuccess && <p role="status">Action logged.</p>}
      <button type="submit" disabled={log.isPending}>
        {log.isPending ? 'Saving…' : 'Log action'}
      </button>
    </form>
  )
}
