import { CalendarIcon, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useInstance } from "@/contexts/InstanceContext";
import { useEditorContext } from "../state";
import { cn } from "@/lib/utils";

const MemoDatePicker = () => {
  const { state, dispatch } = useEditorContext();
  const { memoRelatedSetting } = useInstance();
  const [isOpen, setIsOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Only show if the feature is enabled
  if (!memoRelatedSetting?.enableCustomMemoDate) {
    return null;
  }

  const createTime = state.timestamps.createTime;
  const displayDate = createTime
    ? new Date(createTime).toLocaleDateString()
    : new Date().toLocaleDateString();

  const formatForInput = (d: Date): string => {
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    const hours = String(d.getHours()).padStart(2, "0");
    const minutes = String(d.getMinutes()).padStart(2, "0");
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  };

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.showPicker?.();
    }
  }, [isOpen]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newDate = new Date(e.target.value);
    if (!isNaN(newDate.getTime())) {
      dispatch({
        type: "SET_TIMESTAMPS",
        payload: { createTime: newDate },
      });
    }
  };

  const handleClear = () => {
    dispatch({
      type: "SET_TIMESTAMPS",
      payload: { createTime: undefined },
    });
    setIsOpen(false);
  };

  const handleToggle = () => {
    setIsOpen(!isOpen);
  };

  if (isOpen) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 bg-popover border border-border rounded-lg shadow-sm">
        <CalendarIcon className="w-4 h-4 text-muted-foreground shrink-0" />
        <input
          ref={inputRef}
          type="datetime-local"
          defaultValue={createTime ? formatForInput(new Date(createTime)) : formatForInput(new Date())}
          onChange={handleChange}
          className="text-sm text-foreground bg-transparent outline-none flex-1 min-w-0"
        />
        <button
          type="button"
          onClick={handleClear}
          className="p-1 hover:bg-accent rounded text-muted-foreground hover:text-foreground transition-colors"
          title="Clear custom date"
        >
          <X className="w-4 h-4" />
        </button>
        <button
          type="button"
          onClick={() => setIsOpen(false)}
          className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
        >
          Done
        </button>
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={handleToggle}
      className={cn(
        "flex items-center gap-2 px-3 py-2 text-sm font-medium",
        "bg-secondary hover:bg-secondary/80",
        "text-secondary-foreground",
        "rounded-lg transition-colors",
        "border border-border/50",
      )}
      title="Set custom date for this memo"
    >
      <CalendarIcon className="w-4 h-4" />
      <span>{createTime ? "Custom: " + displayDate : "Today"}</span>
    </button>
  );
};

export default MemoDatePicker;
