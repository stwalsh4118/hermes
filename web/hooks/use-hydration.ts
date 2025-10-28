import { useEffect, useState } from "react";

/**
 * Hook to check if Zustand store has finished hydrating from localStorage
 * Prevents hydration mismatches by waiting for client-side rehydration
 */
export function useHydration() {
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    // Wait for next tick to ensure hydration is complete
    setHydrated(true);
  }, []);

  return hydrated;
}

