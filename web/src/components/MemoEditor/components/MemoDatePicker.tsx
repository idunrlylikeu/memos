import { CalendarIcon, X } from "lucide-react";
import { useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useInstance } from "@/contexts/InstanceContext";
import { cn } from "@/lib/utils";
import { useEditorContext } from "../state";

function formatForInput(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  const hours = String(d.getHours()).padStart(2, "0");
  const minutes = String(d.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function formatDisplayDate(d: Date): string {
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

const MemoDatePicker = () => {
  const { state, dispatch } = useEditorContext();
  const { memoRelatedSetting } = useInstance();
  const [open, setOpen] = useState(false);

  if (!memoRelatedSetting?.enableCustomMemoDate) {
    return null;
  }

  const createTime = state.timestamps.createTime;

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newDate = new Date(e.target.value);
    if (!isNaN(newDate.getTime())) {
      // Only sync updateTime if user hasn't manually set a different updateTime.
      // If updateTime is not set, or it was previously equal to createTime, keep them in sync.
      const currentUpdateTime = state.timestamps.updateTime;
      const shouldSyncUpdate = !currentUpdateTime || (createTime && currentUpdateTime.getTime() === createTime.getTime());

      dispatch({
        type: "SET_TIMESTAMPS",
        payload: shouldSyncUpdate ? { createTime: newDate, updateTime: newDate } : { createTime: newDate },
      });
    }
  };

  const handleClear = () => {
    dispatch({
      type: "SET_TIMESTAMPS",
      payload: { createTime: undefined, updateTime: undefined },
    });
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "flex items-center gap-1.5 px-2.5 py-1.5 text-sm font-medium",
            "bg-secondary hover:bg-secondary/80",
            "text-secondary-foreground",
            "rounded-lg transition-colors",
            "border border-border/50",
          )}
          title="Set custom date for this memo"
        >
          <CalendarIcon className="w-3.5 h-3.5 shrink-0" />
          <span className="hidden sm:inline">{createTime ? formatDisplayDate(createTime) : "Set Date"}</span>
          {createTime && <span className="sm:hidden w-2 h-2 rounded-full bg-primary shrink-0" title={formatDisplayDate(createTime)} />}
        </button>
      </PopoverTrigger>
      <PopoverContent
        side="top"
        align="start"
        className="w-auto p-3 space-y-3"
        // Ensure popover stays within viewport on mobile
        collisionPadding={12}
      >
        <div className="flex items-center justify-between gap-4">
          <span className="text-sm font-medium text-foreground">Custom date</span>
          {createTime && (
            <button
              type="button"
              onClick={handleClear}
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive transition-colors"
              title="Clear custom date"
            >
              <X className="w-3 h-3" />
              Clear
            </button>
          )}
        </div>

        <input
          type="datetime-local"
          defaultValue={createTime ? formatForInput(createTime) : formatForInput(new Date())}
          onChange={handleChange}
          className={cn(
            "block w-full rounded-md border border-border bg-background",
            "px-3 py-2 text-sm text-foreground",
            "focus:outline-none focus:ring-2 focus:ring-ring",
            // Ensure native picker appears on mobile tap (no extra JS needed)
            "cursor-pointer",
          )}
        />

        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => setOpen(false)}
            className="px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
          >
            Done
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
};

export default MemoDatePicker;
