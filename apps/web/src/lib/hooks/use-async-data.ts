import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

interface UseAsyncDataOptions<T> {
  enabled?: boolean;
  initialData: T;
  errorMessage: string;
  autoLoad?: boolean;
}

interface UseAsyncDataResult<T> {
  data: T;
  setData: Dispatch<SetStateAction<T>>;
  loading: boolean;
  error: string | null;
  clearError: () => void;
  refresh: () => Promise<void>;
}

export function useAsyncData<T>(
  fetcher: () => Promise<T>,
  {
    enabled = true,
    initialData,
    errorMessage,
    autoLoad = true,
  }: UseAsyncDataOptions<T>,
): UseAsyncDataResult<T> {
  const [data, setData] = useState<T>(initialData);
  const [loading, setLoading] = useState<boolean>(enabled);
  const [error, setError] = useState<string | null>(null);
  const hasAutoLoadedRef = useRef(false);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetcher();
      setData(result);
    } catch {
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [fetcher, errorMessage]);

  const clearError = useCallback(() => setError(null), []);

  useEffect(() => {
    hasAutoLoadedRef.current = false;
  }, [fetcher]);

  useEffect(() => {
    if (!autoLoad || !enabled) {
      setLoading(false);
      return;
    }
    if (hasAutoLoadedRef.current) return;
    hasAutoLoadedRef.current = true;
    void refresh();
  }, [autoLoad, enabled, refresh]);

  return { data, setData, loading, error, clearError, refresh };
}
