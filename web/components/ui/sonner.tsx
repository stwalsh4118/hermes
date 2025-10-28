"use client"

import {
  CircleCheck,
  Info,
  LoaderCircle,
  OctagonX,
  TriangleAlert,
} from "lucide-react"
import { useTheme } from "next-themes"
import { Toaster as Sonner } from "sonner"

type ToasterProps = React.ComponentProps<typeof Sonner>

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme()

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      closeButton
      icons={{
        success: <CircleCheck className="h-4 w-4" />,
        info: <Info className="h-4 w-4" />,
        warning: <TriangleAlert className="h-4 w-4" />,
        error: <OctagonX className="h-4 w-4" />,
        loading: <LoaderCircle className="h-4 w-4 animate-spin" />,
      }}
      toastOptions={{
        classNames: {
          toast:
            "group toast !bg-card !text-foreground !border-4 !border-primary !shadow-[8px_8px_0_rgba(0,0,0,0.6)] !rounded-lg",
          title: "vcr-text !uppercase !tracking-wider !text-foreground",
          description: "!text-muted-foreground",
          actionButton:
            "!bg-primary !text-primary-foreground !shadow-[4px_4px_0_rgba(0,0,0,0.2)] !rounded-md vcr-text !uppercase !tracking-wider",
          cancelButton:
            "!bg-muted !text-muted-foreground !border-2 !border-primary !shadow-[4px_4px_0_rgba(0,0,0,0.2)] !rounded-md vcr-text !uppercase !tracking-wider",
          closeButton:
            "!bg-card !text-foreground !border-2 !border-primary hover:!bg-muted !shadow-[2px_2px_0_rgba(0,0,0,0.2)]",
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
