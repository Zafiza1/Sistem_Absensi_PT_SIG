import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export type StatTone = "blue" | "emerald" | "amber" | "red";

const TONE_CLASSES: Record<StatTone, string> = {
  blue: "bg-primary/10 text-primary dark:bg-primary/20 dark:text-sky-400",
  emerald: "bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400",
  amber: "bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-400",
  red: "bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-400",
};

const HINT_CLASSES: Record<StatTone, string> = {
  blue: "text-primary",
  emerald: "text-emerald-600 dark:text-emerald-400",
  amber: "text-amber-600 dark:text-amber-400",
  red: "text-red-600 dark:text-red-400",
};

export function StatCard({
  label,
  value,
  icon: Icon,
  tone,
  hint,
  loading,
}: {
  label: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
  tone: StatTone;
  hint?: string;
  loading: boolean;
}) {
  return (
    <Card className="gap-3 p-4">
      <div className="flex items-center gap-3">
        <div className={cn("flex size-11 shrink-0 items-center justify-center rounded-xl", TONE_CLASSES[tone])}>
          <Icon className="size-5" />
        </div>
        <p className="text-sm text-muted-foreground">{label}</p>
      </div>
      {loading ? (
        <Skeleton className="h-7 w-16" />
      ) : (
        <p className="text-2xl font-semibold tracking-tight">{value}</p>
      )}
      {hint && !loading && (
        <p className={cn("text-xs font-medium", HINT_CLASSES[tone])}>{hint}</p>
      )}
    </Card>
  );
}
