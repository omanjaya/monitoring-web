import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

interface UseMutationActionOptions<TData, TVariables> {
  mutationFn: (variables: TVariables) => Promise<TData>
  successMessage?: string
  errorMessage?: string
  invalidateKeys?: string[]
  onSuccess?: (data: TData) => void
}

export function useMutationAction<TData = unknown, TVariables = void>({
  mutationFn,
  successMessage,
  errorMessage = "Terjadi kesalahan",
  invalidateKeys,
  onSuccess,
}: UseMutationActionOptions<TData, TVariables>) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn,
    onSuccess: (data) => {
      if (successMessage) toast.success(successMessage)
      if (invalidateKeys) {
        invalidateKeys.forEach((key) => {
          queryClient.invalidateQueries({ queryKey: [key] })
        })
      }
      onSuccess?.(data)
    },
    onError: (err: Error) => {
      toast.error(err.message || errorMessage)
    },
  })
}
