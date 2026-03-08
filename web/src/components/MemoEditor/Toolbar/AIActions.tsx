import { useState } from "react";
import { BrainCircuitIcon, Loader2Icon, PenLineIcon, FileTextIcon } from "lucide-react";
import toast from "react-hot-toast";

import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { aiService } from "@/utils/aiService";
import { useEditorContext } from "../state";

const AIActions = () => {
    const { state, actions, dispatch } = useEditorContext();
    const [isGenerating, setIsGenerating] = useState(false);

    const handleAIAction = async (systemPrompt: string) => {
        if (!state.content.trim() || isGenerating) return;

        setIsGenerating(true);
        const originalContent = state.content;
        let newContent = "";

        // Clear content before starting stream
        dispatch(actions.updateContent(""));

        try {
            const stream = aiService.streamCompletion(originalContent, systemPrompt);
            for await (const token of stream) {
                newContent += token;
                dispatch(actions.updateContent(newContent));
            }
        } catch (error: any) {
            toast.error(error.message || "Failed to generate AI content");
            dispatch(actions.updateContent(originalContent)); // revert on error
        } finally {
            setIsGenerating(false);
        }
    };

    const handleFormat = () => {
        handleAIAction(`You are an expert Markdown formatting assistant. Your ONLY job is to take the user's text and format it cleanly and beautifully using Markdown. You must not change the meaning, tone, or details of the text. 

CRITICAL INSTRUCTIONS:
- Enhance readability by applying markdown symbols appropriately based on context (e.g. # or ## for headers, -, *, or 1. for lists, **bold** and *italic* for emphasis, \`code\` for inline code, and > for blockquotes).
- Return ONLY the formatted Markdown text.
- Do NOT wrap the text in a Markdown code block (e.g. \`\`\`markdown ... \`\`\`).
- Do NOT add any conversational filler like "Here is the formatted text" or "Sure!".
- JUST RETURN THE FINAL TEXT.`);
    };

    const handleRewrite = () => {
        handleAIAction(`You are an expert writing assistant. Your ONLY job is to rewrite the user's text to make it more concise, clearer, and easier to read, formatted beautifully in Markdown.

CRITICAL INSTRUCTIONS:
- Apply markdown symbols appropriately to structure the result (e.g. # or ## for headers, -, *, or 1. for lists, **bold** and *italic* for emphasis, \`code\` for inline code, and > for blockquotes).
- Return ONLY the rewritten Markdown text.
- Do NOT wrap the text in a Markdown code block (e.g. \`\`\`markdown ... \`\`\`).
- Do NOT add any conversational filler like "Here is the rewritten text" or "Sure!".
- JUST RETURN THE FINAL TEXT.`);
    };

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button
                    size="sm"
                    variant="outline"
                    className="h-8 px-2 flex gap-1"
                    disabled={isGenerating || !state.content.trim()}
                    title="AI Assistant"
                >
                    {isGenerating ? (
                        <Loader2Icon className="w-4 h-4 animate-spin opacity-70" />
                    ) : (
                        <BrainCircuitIcon className="w-4 h-4 opacity-70" />
                    )}
                    <span className="text-xs font-normal hidden sm:inline-block">AI</span>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
                <DropdownMenuItem onClick={handleFormat} className="cursor-pointer">
                    <FileTextIcon className="w-4 h-4 mr-2" />
                    Format Markdown
                </DropdownMenuItem>
                <DropdownMenuItem onClick={handleRewrite} className="cursor-pointer">
                    <PenLineIcon className="w-4 h-4 mr-2" />
                    Rewrite & Format
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    );
};

export default AIActions;
