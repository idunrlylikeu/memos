import { useEditorContext } from "../state";

const PreviewModeToggle = () => {
    const { state, actions, dispatch } = useEditorContext();
    const isRender = state.ui.isPreviewMode;

    const active = "text-primary font-medium hover:text-primary";
    const inactive = "text-muted-foreground hover:text-foreground";

    return (
        <div className="flex items-center gap-3 text-sm px-1 mb-1">
            <button
                type="button"
                className={`${!isRender ? active : inactive} transition-colors`}
                onClick={() => isRender && dispatch(actions.togglePreviewMode())}
                aria-pressed={!isRender}
            >
                MD
            </button>
            <span className="text-muted-foreground/30 text-xs">|</span>
            <button
                type="button"
                className={`${isRender ? active : inactive} transition-colors`}
                onClick={() => !isRender && dispatch(actions.togglePreviewMode())}
                aria-pressed={isRender}
            >
                Render
            </button>
        </div>
    );
};

export default PreviewModeToggle;
