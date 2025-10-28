import React from "react";
import { LoadingSpinner } from "./loading-spinner";
import { ErrorMessage } from "./error-message";
import { EmptyState } from "./empty-state";
import { LucideIcon } from "lucide-react";

interface DataWrapperProps<T> {
  data: T | undefined;
  isLoading: boolean;
  error: Error | null;
  isEmpty?: (data: T) => boolean;
  emptyState?: {
    icon?: LucideIcon;
    title: string;
    description: string;
    action?: {
      label: string;
      onClick: () => void;
    };
  };
  onRetry?: () => void;
  children: (data: T) => React.ReactNode;
}

export function DataWrapper<T>({
  data,
  isLoading,
  error,
  isEmpty,
  emptyState,
  onRetry,
  children,
}: DataWrapperProps<T>) {
  if (isLoading) {
    return <LoadingSpinner />;
  }

  if (error) {
    return (
      <ErrorMessage
        message={error.message || "An error occurred"}
        onRetry={onRetry}
      />
    );
  }

  if (!data || (isEmpty && isEmpty(data))) {
    if (emptyState) {
      return <EmptyState {...emptyState} />;
    }
    return (
      <EmptyState
        title="No data"
        description="There's nothing to display here yet."
      />
    );
  }

  return <>{children(data)}</>;
}

