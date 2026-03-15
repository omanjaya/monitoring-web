import { useQuery } from "@tanstack/react-query"
import { useState, useCallback } from "react"

interface UsePaginatedQueryOptions<TFilters> {
  queryKey: string
  queryFn: (params: string) => Promise<{ data: unknown[]; total: number }>
  pageSize?: number
  initialFilters?: TFilters
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function usePaginatedQuery<TData, TFilters extends Record<string, any> = Record<string, string>>({
  queryKey,
  queryFn,
  pageSize: initialPageSize = 25,
  initialFilters,
}: UsePaginatedQueryOptions<TFilters>) {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(initialPageSize)
  const [filters, setFilters] = useState<TFilters>(initialFilters ?? ({} as TFilters))

  const buildParams = useCallback(() => {
    const params = new URLSearchParams()
    params.set("page", String(page))
    params.set("limit", String(pageSize))
    Object.entries(filters).forEach(([key, value]) => {
      if (value && value !== "all") {
        params.set(key, value)
      }
    })
    return params.toString()
  }, [page, pageSize, filters])

  const query = useQuery({
    queryKey: [queryKey, page, pageSize, filters],
    queryFn: () => queryFn(buildParams()),
  })

  const data = (query.data?.data ?? []) as TData[]
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const setFilter = useCallback((key: keyof TFilters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }))
    setPage(1)
  }, [])

  const setAllFilters = useCallback((newFilters: Partial<TFilters>) => {
    setFilters((prev) => ({ ...prev, ...newFilters }))
    setPage(1)
  }, [])

  const changePageSize = useCallback((newSize: number) => {
    setPageSize(newSize)
    setPage(1)
  }, [])

  return {
    data,
    total,
    totalPages,
    page,
    setPage,
    filters,
    setFilter,
    setAllFilters,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    refetch: query.refetch,
    pageSize,
    setPageSize: changePageSize,
  }
}
