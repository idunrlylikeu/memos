# TypeScript Type Organization Guide

How to structure your types for clarity, maintainability, and scalability.

---

## Table of Contents

1. [Overview: How Memos Organizes Types](#1-overview-how-memos-organizes-types)
2. [Pattern 1: Centralized Types Folder](#2-pattern-1-centralized-types-folder)
3. [Pattern 2: Co-located Types](#3-pattern-2-co-located-types)
4. [Pattern 3: Barrel Exports](#4-pattern-3-barrel-exports)
5. [Pattern 4: Domain-Driven Types](#5-pattern-4-domain-driven-types)
6. [When to Use Each Pattern](#6-when-to-use-each-pattern)
7. [Best Practices](#7-best-practices)

---

## 1. Overview: How Memos Organizes Types

Memos uses a **hybrid approach** combining multiple patterns:

```
web/src/
├── types/                          # Centralized types
│   ├── common.ts                   # Shared types
│   ├── markdown.ts                 # Domain-specific
│   ├── statistics.ts               # Domain-specific
│   ├── proto/                      # Auto-generated (don't edit)
│   │   └── api/v1/
│   │       ├── memo_service_pb.ts
│   │       ├── user_service_pb.ts
│   │       └── ...
│   └── modules/
│       └── setting.d.ts            # Module augmentation
│
└── components/
    └── MemoEditor/
        └── types/                  # Component-specific types
            ├── attachment.ts       # Attachment types + utils
            ├── components.ts       # Component props
            ├── context.ts          # Context types
            ├── insert-menu.ts      # Feature-specific
            └── index.ts            # Barrel export
```

---

## 2. Pattern 1: Centralized Types Folder

### When to Use
- **Shared types** used across multiple features
- **API/response types** from backend
- **Common utilities** used globally

### Folder Structure

```
src/
└── types/
    ├── common.ts              # Generic reusable types
    ├── api.ts                 # API request/response types
    ├── components.ts          # Shared component prop types
    └── domain/
        ├── user.ts            # User-related types
        ├── memo.ts            # Memo-related types
        └── auth.ts            # Auth-related types
```

### Example: `types/common.ts`

**Real example from Memos** (`web/src/types/common.ts`):

```typescript
// ✅ Simple, focused types
export type TableData = Record<string, unknown>;

export interface ApiError {
  message: string;
  code?: string;
  details?: unknown;
}

// Type guard function
export function isApiError(error: unknown): error is ApiError {
  return typeof error === "object" &&
         error !== null &&
         "message" in error &&
         typeof (error as ApiError).message === "string";
}

export type ToastFunction = (message: string) => void | Promise<void>;
```

### Example: `types/domain/user.ts`

```typescript
// Domain-specific user types
export interface User {
  id: string;
  name: string;
  email: string;
  role: UserRole;
}

export type UserRole = "admin" | "user" | "guest";

export interface UserStats {
  memoCount: number;
  tagCount: Record<string, number>;
}

// Factory function
export function createUser(data: Partial<User>): User {
  return {
    id: data.id ?? "",
    name: data.name ?? "",
    email: data.email ?? "",
    role: data.role ?? "user",
  };
}
```

### Pros
- ✅ Easy to find shared types
- ✅ Prevents circular dependencies
- ✅ Clear separation of concerns

### Cons
- ❌ Can become a "dumping ground"
- ❌ More imports: `import type { User } from "@/types/domain/user"`
- ❌ Harder to find component-specific types

---

## 3. Pattern 2: Co-located Types

### When to Use
- **Component-specific props**
- **Feature-specific state**
- Types used in **only one place**

### Folder Structure

```
src/
└── components/
    └── MemoEditor/
        ├── index.tsx              # Main component
        ├── Editor.tsx
        ├── state/
        │   ├── reducer.ts
        │   └── types.ts          # State types (co-located)
        └── types/
            ├── attachment.ts     # Attachment-related
            ├── context.ts        # Context types
            └── index.ts          # Barrel export
```

### Example: Component with Co-located Types

**Real example from Memos** (`web/src/components/MemoEditor/state/types.ts`):

```typescript
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import type { Location, MemoRelation } from "@/types/proto/api/v1/memo_service_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import type { LocalFile } from "../types/attachment";

// State interface for the editor
export interface EditorState {
  content: string;
  metadata: {
    visibility: Visibility;
    attachments: Attachment[];
    relations: MemoRelation[];
    location?: Location;
  };
  ui: {
    isFocusMode: boolean;
    isLoading: {
      saving: boolean;
      uploading: boolean;
      loading: boolean;
    };
    isDragging: boolean;
    isComposing: boolean;
  };
  timestamps: {
    createTime?: Date;
    updateTime?: Date;
  };
  localFiles: LocalFile[];
}

// Discriminated union for actions
export type EditorAction =
  | { type: "INIT_MEMO"; payload: { content: string; metadata: EditorState["metadata"]; timestamps: EditorState["timestamps"] } }
  | { type: "UPDATE_CONTENT"; payload: string }
  | { type: "SET_METADATA"; payload: Partial<EditorState["metadata"]> }
  | { type: "ADD_ATTACHMENT"; payload: Attachment }
  | { type: "REMOVE_ATTACHMENT"; payload: string }
  | { type: "ADD_RELATION"; payload: MemoRelation }
  | { type: "REMOVE_RELATION"; payload: string }
  | { type: "ADD_LOCAL_FILE"; payload: LocalFile }
  | { type: "REMOVE_LOCAL_FILE"; payload: string }
  | { type: "CLEAR_LOCAL_FILES" }
  | { type: "SET_TIMESTAMPS"; payload: Partial<EditorState["timestamps"]> }
  | { type: "TOGGLE_FOCUS_MODE" }
  | { type: "SET_LOADING"; payload: { key: LoadingKey; value: boolean } }
  | { type: "SET_DRAGGING"; payload: boolean }
  | { type: "SET_COMPOSING"; payload: boolean }
  | { type: "RESET" };

export type LoadingKey = "saving" | "uploading" | "loading";

// Initial state
export const initialState: EditorState = {
  content: "",
  metadata: {
    visibility: Visibility.PRIVATE,
    attachments: [],
    relations: [],
    location: undefined,
  },
  ui: {
    isFocusMode: false,
    isLoading: {
      saving: false,
      uploading: false,
      loading: false,
    },
    isDragging: false,
    isComposing: false,
  },
  timestamps: {
    createTime: undefined,
    updateTime: undefined,
  },
  localFiles: [],
};
```

### Example: Feature-specific Types with Utils

**Real example from Memos** (`web/src/components/MemoEditor/types/attachment.ts`):

```typescript
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { getAttachmentThumbnailUrl, getAttachmentType, getAttachmentUrl } from "@/utils/attachment";

// Type definition
export type FileCategory = "image" | "video" | "document";

// Unified view model for rendering
export interface AttachmentItem {
  readonly id: string;
  readonly filename: string;
  readonly category: FileCategory;
  readonly mimeType: string;
  readonly thumbnailUrl: string;
  readonly sourceUrl: string;
  readonly size?: number;
  readonly isLocal: boolean;
}

// Local files being uploaded
export interface LocalFile {
  readonly file: File;
  readonly previewUrl: string;
}

// Helper function
function categorizeFile(mimeType: string): FileCategory {
  if (mimeType.startsWith("image/")) return "image";
  if (mimeType.startsWith("video/")) return "video";
  return "document";
}

// Conversion functions
export function attachmentToItem(attachment: Attachment): AttachmentItem {
  const attachmentType = getAttachmentType(attachment);
  const sourceUrl = getAttachmentUrl(attachment);

  return {
    id: attachment.name,
    filename: attachment.filename,
    category: categorizeFile(attachment.type),
    mimeType: attachment.type,
    thumbnailUrl: attachmentType === "image/*" ? getAttachmentThumbnailUrl(attachment) : sourceUrl,
    sourceUrl,
    size: Number(attachment.size),
    isLocal: false,
  };
}

export function fileToItem(file: File, blobUrl: string): AttachmentItem {
  return {
    id: blobUrl,
    filename: file.name,
    category: categorizeFile(file.type),
    mimeType: file.type,
    thumbnailUrl: blobUrl,
    sourceUrl: blobUrl,
    size: file.size,
    isLocal: true,
  };
}

// Utility functions
export function toAttachmentItems(
  attachments: Attachment[],
  localFiles: LocalFile[] = []
): AttachmentItem[] {
  return [
    ...attachments.map(attachmentToItem),
    ...localFiles.map(({ file, previewUrl }) => fileToItem(file, previewUrl))
  ];
}

export function filterByCategory(
  items: AttachmentItem[],
  categories: FileCategory[]
): AttachmentItem[] {
  const categorySet = new Set(categories);
  return items.filter((item) => categorySet.has(item.category));
}

export function separateMediaAndDocs(
  items: AttachmentItem[]
): { media: AttachmentItem[]; docs: AttachmentItem[] } {
  const media: AttachmentItem[] = [];
  const docs: AttachmentItem[] = [];

  for (const item of items) {
    if (item.category === "image" || item.category === "video") {
      media.push(item);
    } else {
      docs.push(item);
    }
  }

  return { media, docs };
}
```

### Pros
- ✅ Types are **right next to** where they're used
- ✅ Easy to update types when changing components
- ✅ Clear what types belong to which feature
- ✅ Can include related helper functions

### Cons
- ❌ Harder to share types across components
- ❌ Can lead to duplicate type definitions

---

## 4. Pattern 3: Barrel Exports

### When to Use
- Organizing imports from multiple type files
- Creating a clean public API for a module

### How It Works

Instead of:
```typescript
// ❌ Messy imports
import type { EditorProps } from "@/components/MemoEditor/types/components";
import type { MemoEditorContextValue } from "@/components/MemoEditor/types/context";
import type { LocationState } from "@/components/MemoEditor/types/insert-menu";
```

Use a barrel file (`index.ts`):
```typescript
// ✅ Clean import
import type {
  EditorProps,
  MemoEditorContextValue,
  LocationState
} from "@/components/MemoEditor/types";
```

### Example: Barrel Export

**Real example from Memos** (`web/src/components/MemoEditor/types/index.ts`):

```typescript
// MemoEditor type exports

// Component props
export type {
  EditorContentProps,
  EditorMetadataProps,
  EditorProps,
  EditorToolbarProps,
  FocusModeExitButtonProps,
  FocusModeOverlayProps,
  InsertMenuProps,
  LinkMemoDialogProps,
  LocationDialogProps,
  MemoEditorProps,
  SlashCommandsProps,
  TagSuggestionsProps,
  VisibilitySelectorProps,
} from "./components";

// Context types
export { MemoEditorContext, type MemoEditorContextValue } from "./context";

// Feature types
export type { LocationState } from "./insert-menu";
```

### Folder Structure with Barrel

```
components/
└── MemoEditor/
    ├── types/
    │   ├── attachment.ts      # Defines AttachmentItem, LocalFile
    │   ├── components.ts      # Defines all component props
    │   ├── context.ts         # Defines context types
    │   ├── insert-menu.ts     # Defines insert menu state
    │   └── index.ts           # Re-exports everything
    ├── MemoEditor.tsx
    ├── Editor.tsx
    └── Toolbar.tsx
```

### Usage in Components

```typescript
// components/MemoEditor/MemoEditor.tsx
import type { MemoEditorProps } from "./types";

// components/MemoEditor/Toolbar/VisibilitySelector.tsx
import type { VisibilitySelectorProps } from "../../types";

// components/MemoEditor/InsertMenu.tsx
import type { LocationState } from "../types";
```

### Pro Tip: Named Exports Only

```typescript
// ✅ Good - Named exports
export type { User, UserRole, UserStats } from "./user";
export type { Memo, MemoState } from "./memo";

// ❌ Avoid - Default exports (harder to refactor)
export type { default as UserTypes } from "./user";
```

---

## 5. Pattern 4: Domain-Driven Types

### When to Use
- **Large applications** with clear domain boundaries
- **Micro-frontends** or feature-based architecture
- **Team collaboration** (different teams own different domains)

### Folder Structure

```
src/
├── types/
│   └── common.ts                # Cross-domain types
│
├── features/
│   ├── auth/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── types/
│   │   │   ├── index.ts         # Barrel export
│   │   │   ├── user.ts          # User types
│   │   │   ├── session.ts       # Session types
│   │   │   └── permissions.ts   # Permission types
│   │   └── api.ts
│   │
│   ├── memos/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── types/
│   │   │   ├── index.ts
│   │   │   ├── memo.ts
│   │   │   ├── tag.ts
│   │   │   └── reaction.ts
│   │   └── api.ts
│   │
│   └── settings/
│       ├── components/
│       ├── hooks/
│       ├── types/
│       │   ├── index.ts
│       │   ├── user-settings.ts
│       │   └── instance-settings.ts
│       └── api.ts
```

### Example: Domain Type Structure

```typescript
// features/auth/types/user.ts
export interface User {
  id: string;
  username: string;
  email: string;
  role: Role;
}

export type Role = "admin" | "user" | "guest";

// features/auth/types/session.ts
export interface Session {
  user: User;
  token: string;
  expiresAt: Date;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

// features/auth/types/permissions.ts
export type Permission = string;

export interface PermissionCheck {
  resource: string;
  action: "read" | "write" | "delete";
}

export function hasPermission(
  user: User,
  check: PermissionCheck
): boolean {
  // Implementation
}

// features/auth/types/index.ts
export type { User, Role } from "./user";
export type { Session, LoginCredentials } from "./session";
export type { Permission, PermissionCheck, hasPermission } from "./permissions";
```

### Usage Across Features

```typescript
// features/memos/components/MemoList.tsx
import type { User } from "@/features/auth/types";  // Cross-feature import

export function MemoList({ createdBy }: { createdBy: User }) {
  // ...
}
```

---

## 6. When to Use Each Pattern

### Decision Tree

```
Where does the type live?

┌─ Is it auto-generated (protobuf, openapi)?
│  └─► Put in types/generated/ or types/proto/
│     Example: types/proto/api/v1/memo_service_pb.ts
│
├─ Is it shared across 3+ unrelated features?
│  └─► Put in types/domain/ or types/common.ts
│     Example: types/common.ts (ApiError, ToastFunction)
│
├─ Is it only used within one component/feature?
│  ├─► Put in ComponentName/types/
│  │   Example: components/MemoEditor/types/attachment.ts
│  │
│  └─► Or co-locate with the file
│      Example: components/MemoEditor/state/types.ts
│
└─ Is it specific to a feature domain?
   └─► Put in features/FeatureName/types/
       Example: features/auth/types/user.ts
```

### Quick Reference

| Pattern | Use When | Example |
|---------|----------|---------|
| **Centralized** | Shared types, API types | `types/common.ts` |
| **Co-located** | Component-specific | `MemoEditor/types/attachment.ts` |
| **Barrel export** | Organizing imports | `types/index.ts` |
| **Domain-driven** | Large apps, feature teams | `features/auth/types/` |
| **Auto-generated** | Protobuf, OpenAPI | `types/proto/` |

---

## 7. Best Practices

### 7.1 Type vs Interface

**Use `interface` when:**
- Defining object shapes
- You might extend the type later
- The type is public API

```typescript
// ✅ Interface for public API
export interface UserProps {
  user: User;
  onEdit: (user: User) => void;
}

// Can extend later
export interface AdminUserProps extends UserProps {
  permissions: Permission[];
}
```

**Use `type` when:**
- Creating unions, intersections, or mapped types
- Working with utility types
- The type is private/internal

```typescript
// ✅ Type for unions and utilities
export type UserRole = "admin" | "user" | "guest";
export type PartialUser = Partial<User>;
export type UserById = Record<string, User>;
```

### 7.2 Export Type Only

```typescript
// ✅ Clear - this is a type export
export type { User, UserRole } from "./user";

// ✅ Also clear
export type { User } from "./user";
export type { UserRole } from "./user";

// ❌ Confusing - value or type?
export { User, UserRole } from "./user";
```

### 7.3 Avoid Circular Dependencies

```typescript
// ❌ Bad: Circular dependency
// types/user.ts
import type { Memo } from "./memo";
export interface User { memos: Memo[]; }

// types/memo.ts
import type { User } from "./user";
export interface Memo { author: User; }

// ✅ Good: Shared types file
// types/common.ts
export interface UserPreview { id: string; name: string; }
export interface MemoPreview { id: string; title: string; }

// types/user.ts
import type { MemoPreview } from "./common";
export interface User { memos: MemoPreview[]; }

// types/memo.ts
import type { UserPreview } from "./common";
export interface Memo { author: UserPreview; }
```

### 7.4 Type Guards for Validation

```typescript
// types/user.ts
export interface User {
  id: string;
  name: string;
  email: string;
}

// Type guard - validates at runtime
export function isUser(value: unknown): value is User {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    "name" in value &&
    "email" in value &&
    typeof value.id === "string" &&
    typeof value.name === "string" &&
    typeof value.email === "string"
  );
}

// Usage
function processUser(data: unknown) {
  if (isUser(data)) {
    // TypeScript knows this is User
    console.log(data.name.toUpperCase());
  }
}
```

### 7.5 Re-export External Types

```typescript
// ❌ Bad: Everyone imports from protobuf directly
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";
import type { User } from "@/types/proto/api/v1/user_service_pb";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";

// ✅ Good: Centralized re-exports
// types/api.ts
export type { Memo } from "./proto/api/v1/memo_service_pb";
export type { User } from "./proto/api/v1/user_service_pb";
export type { Attachment } from "./proto/api/v1/attachment_service_pb";

// ✅ Usage: Much cleaner
import type { Memo, User, Attachment } from "@/types/api";
```

### 7.6 Naming Conventions

| Pattern | Name | Example |
|---------|------|---------|
| Props interface | `ComponentNameProps` | `UserCardProps` |
| State interface | `FeatureNameState` | `EditorState` |
| Action type | `FeatureNameAction` | `EditorAction` |
| Context value | `FeatureNameContextValue` | `AuthContextValue` |
| Type guard | `isTypeName` | `isApiError`, `isUser` |
| Factory function | `createTypeName` | `createUser` |
| Type alias | PascalCase | `UserRole`, `FileCategory` |

### 7.7 File Naming

```
types/
├── common.ts              # ✅ Generic shared types
├── api.ts                 # ✅ API-related types
├── markdown.ts            # ✅ Domain-specific
├── user.types.ts          # ❌ Avoid redundant suffix
├── userTypes.ts           # ❌ Avoid camelCase
└── user.ts                # ✅ Use PascalCase for type files
```

### 7.8 Organize Related Types

```typescript
// ✅ Good: Grouped by purpose
// types/user.ts

// Core types
export interface User {
  id: string;
  name: string;
  email: string;
}

export type UserRole = "admin" | "user" | "guest";

// Derived types
export type UserPreview = Pick<User, "id" | "name">;
export type UserUpdate = Partial<Omit<User, "id">>;

// Collections
export type UserById = Record<string, User>;
export type UsersByRole = Partial<Record<UserRole, User[]>>;

// Props
export interface UserCardProps {
  user: User;
  onEdit?: (user: User) => void;
}

export interface UserListProps {
  users: User[];
  loading?: boolean;
}

// Guards
export function isValidUserRole(value: string): value is UserRole {
  return ["admin", "user", "guest"].includes(value);
}

// Factories
export function createUser(data: Partial<User>): User {
  return {
    id: data.id ?? crypto.randomUUID(),
    name: data.name ?? "",
    email: data.email ?? "",
  };
}
```

### 7.9 Comment Your Types

```typescript
// ✅ Self-documenting with comments

/**
 * Represents a user in the system.
 * @remarks Users can have one of three roles with different permissions.
 */
export interface User {
  /** Unique identifier (format: users/{id}) */
  id: string;

  /** Display name (2-50 characters) */
  name: string;

  /** Email address (must be valid format) */
  email: string;

  /** User's role determines their permissions */
  role: UserRole;

  /** When the user was created */
  createdAt: Date;

  /** Last time user was active */
  lastActiveAt?: Date;  // Optional - can be null
}

/**
 * User roles with hierarchical permissions.
 * - admin: Full system access
 * - user: Standard user permissions
 * - guest: Read-only access
 */
export type UserRole = "admin" | "user" | "guest";
```

---

## 8. Complete Example: E-Commerce App

### Folder Structure

```
src/
├── types/
│   ├── common.ts              # Shared types
│   ├── api.ts                 # API re-exports
│   └── index.ts               # Main barrel
│
├── features/
│   ├── products/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── types/
│   │   │   ├── index.ts       # Barrel export
│   │   │   ├── product.ts     # Product types
│   │   │   ├── category.ts    # Category types
│   │   │   └── inventory.ts   # Inventory types
│   │   └── api.ts
│   │
│   ├── cart/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── types/
│   │   │   ├── index.ts
│   │   │   └── cart.ts
│   │   └── api.ts
│   │
│   └── checkout/
│       ├── components/
│       ├── hooks/
│       ├── types/
│       │   ├── index.ts
│       │   └── checkout.ts
│       └── api.ts
│
└── components/
    └── shared/
        ├── Button/
        │   ├── Button.tsx
        │   └── types.ts       # Component-specific types
        └── Modal/
            ├── Modal.tsx
            └── types.ts
```

### Type Files

```typescript
// types/common.ts
export type EntityId = string;

export interface Timestamps {
  createdAt: Date;
  updatedAt: Date;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ApiError {
  message: string;
  code?: string;
  details?: unknown;
}

export function isApiError(error: unknown): error is ApiError {
  return (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof (error as ApiError).message === "string"
  );
}
```

```typescript
// features/products/types/product.ts
import type { EntityId, Timestamps } from "@/types/common";

export type ProductStatus = "draft" | "active" | "archived";

export interface Product extends Timestamps {
  id: EntityId;
  name: string;
  description: string;
  price: number;
  status: ProductStatus;
  categories: Category[];
  images: ProductImage[];
}

export interface ProductImage {
  id: EntityId;
  url: string;
  altText?: string;
  isPrimary: boolean;
}

export interface ProductPreview {
  id: EntityId;
  name: string;
  price: number;
  primaryImage?: string;
}

// Type transformations
export type ProductUpdate = Partial<Omit<Product, "id" | "createdAt" | "updatedAt">>;

export type ProductDraft = Omit<Product, "id" | "createdAt" | "updatedAt">;
```

```typescript
// features/cart/types/cart.ts
import type { EntityId } from "@/types/common";
import type { ProductPreview } from "@/features/products/types/product";

export interface CartItem {
  productId: EntityId;
  product: ProductPreview;
  quantity: number;
}

export interface Cart {
  id: EntityId;
  items: CartItem[];
  total: number;
  currency: string;
}

export interface CartContextValue {
  cart: Cart | null;
  addItem: (product: ProductPreview, quantity?: number) => void;
  removeItem: (productId: EntityId) => void;
  updateQuantity: (productId: EntityId, quantity: number) => void;
  clearCart: () => void;
}
```

```typescript
// features/products/types/index.ts (Barrel)
export type { Product, ProductStatus, ProductImage, ProductPreview, ProductUpdate, ProductDraft } from "./product";
export type { Category, CategoryTree } from "./category";
export type { Inventory, StockLevel } from "./inventory";
```

---

## 9. Quick Checklist

When creating a new type, ask yourself:

- [ ] **Is it shared?** → Put in `types/common.ts`
- [ ] **Is it feature-specific?** → Put in `features/FeatureName/types/`
- [ ] **Is it component-specific?** → Put in `ComponentName/types/`
- [ ] **Is it auto-generated?** → Put in `types/generated/`
- [ ] **Do I need a barrel export?** → Create `index.ts`
- [ ] **Should I use `interface` or `type`?** → Use `interface` for object shapes, `type` for unions/utilities
- [ ] **Do I need a type guard?** → Add `isTypeName()` function
- [ ] **Is the naming clear?** → Follow naming conventions

---

## 10. Key Takeaways

1. **Centralized** (`types/`) for shared, cross-feature types
2. **Co-located** (`ComponentName/types/`) for component-specific types
3. **Barrel exports** (`index.ts`) for clean imports
4. **Domain-driven** (`features/FeatureName/types/`) for large apps
5. **Auto-generated** types go in isolated folder
6. **Type guards** for runtime validation
7. **Re-export** external types for cleaner imports
8. **Comment** complex types for clarity
9. **Organize** by purpose, not by type
10. **Use `interface`** for shapes, **`type`** for unions/utilities
