import {
    BoldIcon,
    CodeIcon,
    Heading1Icon,
    Heading2Icon,
    Heading3Icon,
    ItalicIcon,
    LinkIcon,
    ListIcon,
    ListOrderedIcon,
    MinusIcon,
    QuoteIcon,
    SquareCheckIcon,
    StrikethroughIcon,
} from "lucide-react";
import type { RefObject } from "react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { EditorRefActions } from "../Editor";

interface MarkdownToolbarProps {
    editorRef: RefObject<EditorRefActions | null>;
}

interface ToolbarItem {
    key: string;
    label: string;
    icon: React.ElementType;
    action: (editor: EditorRefActions) => void;
}

function wrapOrInsert(editor: EditorRefActions, prefix: string, suffix: string, placeholder: string) {
    const selected = editor.getSelectedContent();
    if (selected) {
        editor.insertText("", prefix, suffix);
    } else {
        const pos = editor.getCursorPosition();
        editor.insertText(placeholder, prefix, suffix);
        // select the placeholder text so user can type over it
        editor.setCursorPosition(pos + prefix.length, pos + prefix.length + placeholder.length);
    }
}

function prefixLine(editor: EditorRefActions, linePrefix: string) {
    const lineNum = editor.getCursorLineNumber();
    const line = editor.getLine(lineNum);
    if (!line.startsWith(linePrefix)) {
        editor.setLine(lineNum, linePrefix + line);
    }
    editor.focus();
}

const TOOLBAR_ITEMS: ToolbarItem[] = [
    {
        key: "bold",
        label: "Bold",
        icon: BoldIcon,
        action: (e) => wrapOrInsert(e, "**", "**", "bold text"),
    },
    {
        key: "italic",
        label: "Italic",
        icon: ItalicIcon,
        action: (e) => wrapOrInsert(e, "*", "*", "italic text"),
    },
    {
        key: "strikethrough",
        label: "Strikethrough",
        icon: StrikethroughIcon,
        action: (e) => wrapOrInsert(e, "~~", "~~", "strikethrough"),
    },
    {
        key: "code",
        label: "Inline code",
        icon: CodeIcon,
        action: (e) => wrapOrInsert(e, "`", "`", "code"),
    },
    {
        key: "h1",
        label: "Heading 1",
        icon: Heading1Icon,
        action: (e) => prefixLine(e, "# "),
    },
    {
        key: "h2",
        label: "Heading 2",
        icon: Heading2Icon,
        action: (e) => prefixLine(e, "## "),
    },
    {
        key: "h3",
        label: "Heading 3",
        icon: Heading3Icon,
        action: (e) => prefixLine(e, "### "),
    },
    {
        key: "blockquote",
        label: "Blockquote",
        icon: QuoteIcon,
        action: (e) => prefixLine(e, "> "),
    },
    {
        key: "hr",
        label: "Horizontal rule",
        icon: MinusIcon,
        action: (e) => e.insertText("\n\n---\n\n"),
    },
    {
        key: "bullet",
        label: "Bullet list",
        icon: ListIcon,
        action: (e) => prefixLine(e, "- "),
    },
    {
        key: "ordered",
        label: "Numbered list",
        icon: ListOrderedIcon,
        action: (e) => prefixLine(e, "1. "),
    },
    {
        key: "task",
        label: "Task item",
        icon: SquareCheckIcon,
        action: (e) => prefixLine(e, "- [ ] "),
    },
    {
        key: "link",
        label: "Link",
        icon: LinkIcon,
        action: (e) => {
            const selected = e.getSelectedContent();
            if (selected) {
                e.insertText("", "[", "](url)");
            } else {
                const pos = e.getCursorPosition();
                e.insertText("text](url", "[", ")");
                e.setCursorPosition(pos + 1, pos + 5);
            }
        },
    },
];

const SEPARATOR_AFTER = new Set(["code", "blockquote", "hr"]);

const MarkdownToolbar = ({ editorRef }: MarkdownToolbarProps) => {
    return (
        <div className="flex flex-row flex-wrap items-center gap-0.5 py-1 px-0.5 border-b border-border/50">
            {TOOLBAR_ITEMS.map((item) => (
                <>
                    <Tooltip key={item.key}>
                        <TooltipTrigger asChild>
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-foreground"
                                onMouseDown={(e) => {
                                    // prevent textarea from losing focus
                                    e.preventDefault();
                                    const editor = editorRef.current;
                                    if (editor) item.action(editor);
                                }}
                            >
                                <item.icon className="size-3.5" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs">
                            {item.label}
                        </TooltipContent>
                    </Tooltip>
                    {SEPARATOR_AFTER.has(item.key) && (
                        <Separator key={`sep-${item.key}`} orientation="vertical" className="mx-0.5 h-4" />
                    )}
                </>
            ))}
        </div>
    );
};

export default MarkdownToolbar;
