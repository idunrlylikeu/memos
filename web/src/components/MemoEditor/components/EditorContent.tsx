import { forwardRef, useCallback } from "react";
import MemoContent from "@/components/MemoContent";
import Editor, { type EditorRefActions } from "../Editor";
import { useBlobUrls, useDragAndDrop } from "../hooks";
import { useEditorContext } from "../state";
import MarkdownToolbar from "../Toolbar/MarkdownToolbar";
import PreviewModeToggle from "../Toolbar/PreviewModeToggle";
import type { EditorContentProps } from "../types";
import type { LocalFile } from "../types/attachment";

export const EditorContent = forwardRef<EditorRefActions, EditorContentProps>(({ placeholder }, ref) => {
  const { state, actions, dispatch } = useEditorContext();
  const { createBlobUrl } = useBlobUrls();

  const { dragHandlers } = useDragAndDrop((files: FileList) => {
    const localFiles: LocalFile[] = Array.from(files).map((file) => ({
      file,
      previewUrl: createBlobUrl(file),
    }));
    localFiles.forEach((localFile) => dispatch(actions.addLocalFile(localFile)));
  });

  const handleCompositionStart = () => {
    dispatch(actions.setComposing(true));
  };

  const handleCompositionEnd = () => {
    dispatch(actions.setComposing(false));
  };

  const handleContentChange = (content: string) => {
    dispatch(actions.updateContent(content));
  };

  const handlePaste = (event: React.ClipboardEvent<Element>) => {
    const clipboard = event.clipboardData;
    if (!clipboard) return;

    const files: File[] = [];
    if (clipboard.items && clipboard.items.length > 0) {
      for (const item of Array.from(clipboard.items)) {
        if (item.kind !== "file") continue;
        const file = item.getAsFile();
        if (file) files.push(file);
      }
    } else if (clipboard.files && clipboard.files.length > 0) {
      files.push(...Array.from(clipboard.files));
    }

    if (files.length === 0) return;

    const localFiles: LocalFile[] = files.map((file) => ({
      file,
      previewUrl: createBlobUrl(file),
    }));
    localFiles.forEach((localFile) => dispatch(actions.addLocalFile(localFile)));
    event.preventDefault();
  };

  const isRenderMode = state.ui.isPreviewMode;

  const switchToEditMode = useCallback(() => {
    dispatch(actions.togglePreviewMode());
    // Use setTimeout so the Editor has a chance to mount before we focus it
    setTimeout(() => {
      if (typeof ref === "object" && ref?.current) {
        ref.current.focus();
      }
    }, 0);
  }, [dispatch, actions, ref]);

  return (
    <div className="w-full flex flex-col flex-1" {...dragHandlers}>
      <PreviewModeToggle />

      {/* Markdown formatting toolbar — only visible in MD mode */}
      {!isRenderMode && <MarkdownToolbar editorRef={ref as React.RefObject<EditorRefActions | null>} />}

      {isRenderMode ? (
        <div
          className="w-full min-h-[4rem] py-2 px-1 cursor-text"
          onClick={switchToEditMode}
          title="Click to edit"
        >
          {state.content ? (
            <MemoContent content={state.content} />
          ) : (
            <p className="text-muted-foreground opacity-70 text-base">{placeholder}</p>
          )}
        </div>
      ) : (
        <Editor
          ref={ref}
          className="memo-editor-content"
          initialContent={state.content}
          placeholder={placeholder || ""}
          isFocusMode={state.ui.isFocusMode}
          isInIME={state.ui.isComposing}
          onContentChange={handleContentChange}
          onPaste={handlePaste}
          onCompositionStart={handleCompositionStart}
          onCompositionEnd={handleCompositionEnd}
        />
      )}
    </div>
  );
});

EditorContent.displayName = "EditorContent";
