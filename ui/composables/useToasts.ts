// Thin wrapper over Nuxt UI's toast system so handlers don't import the
// component every time and so we have a consistent vocabulary
// (success / error / info).

export function useToasts() {
  const toast = useToast()
  return {
    success: (title: string, description?: string) =>
      toast.add({ title, description, color: 'success', icon: 'i-lucide-check' }),
    error: (title: string, description?: string) =>
      toast.add({ title, description, color: 'error', icon: 'i-lucide-circle-alert' }),
    info: (title: string, description?: string) =>
      toast.add({ title, description, icon: 'i-lucide-info' }),
  }
}

/** Extract a human-readable message from a thrown $fetch error. */
export function extractError(err: any): string {
  if (typeof err === 'string') return err
  return (
    err?.data?.error ??
    err?.response?._data?.error ??
    err?.message ??
    'Algo salió mal.'
  )
}
