# Complete TSX, React Query & Backend Architecture Guide

A comprehensive guide to understanding TypeScript/TSX, React Query, and the Go/gRPC backend stack using the Memos codebase as reference.

---

## Table of Contents

1. [TypeScript Fundamentals](#1-typescript-fundamentals)
2. [TSX & React Patterns](#2-tsx--react-patterns)
3. [TypeScript Utility Types & Keywords](#3-typescript-utility-types--keywords)
4. [React Query Deep Dive](#4-react-query-deep-dive)
5. [Backend Stack: Go + gRPC vs Node/Express vs FastAPI](#5-backend-stack-go--grpc-vs-nodeexpress-vs-fastapi)
6. [Frontend-Backend Communication](#6-frontend-backend-communication)
7. [State Management Patterns](#7-state-management-patterns)

---

## 1. TypeScript Fundamentals

### 1.1 What is TypeScript?

TypeScript is JavaScript with static typing. Instead of:

```javascript
// JavaScript - no type checking
function getUser(id) {
  return { id: id, name: "John" };
}

const user = getUser(1);
user.name.toUpperCase(); // Works
user.age.toFixed(); // Runtime error if age doesn't exist!
```

TypeScript enforces types at compile time:

```typescript
// TypeScript - types checked before code runs
interface User {
  id: number;
  name: string;
  age?: number;  // Optional property
}

function getUser(id: number): User {
  return { id: id, name: "John" };
}

const user = getUser(1);
user.name.toUpperCase(); // ✅ Works - TypeScript knows this is a string
user.age.toFixed();      // ❌ Compile error - age might be undefined!
```

**Real example from Memos** (`web/src/types/proto/api/v1/memo_service_pb.ts:79-203`):

```typescript
export type Memo = Message<"memos.api.v1.Memo"> & {
  // Required field - MUST be present
  name: string;

  // Optional field - marked with ? (can be undefined)
  createTime?: Timestamp;

  // Array of strings
  tags: string[];

  // Boolean field
  pinned: boolean;

  // Nested object (also optional)
  location?: Location;
};
```

### 1.2 Type Annotations vs Type Inference

**Type Annotation** - you explicitly tell TypeScript the type:

```typescript
// Explicit type annotation
const pageSize: number = 50;
const visibility: Visibility = Visibility.PRIVATE;
```

**Type Inference** - TypeScript figures it out automatically:

```typescript
// TypeScript infers this is a number
const pageSize = 50;

// TypeScript infers this is an array of strings
const tags = ["react", "typescript", "go"];
```

**Real example** (`web/src/hooks/useMemoQueries.ts:19-27`):

```typescript
// Type annotation on function parameter
export function useMemos(request: Partial<ListMemosRequest> = {}) {
  return useQuery({
    queryKey: memoKeys.list(request),
    queryFn: async () => {
      const response = await memoServiceClient.listMemos(/* ... */);
      return response; // TypeScript infers return type from useQuery
    },
  });
}
```

---

## 2. TSX & React Patterns

### 2.1 What is TSX?

TSX = TypeScript + JSX. It's the same JSX you know, but with type checking.

**React.FC** - Function Component type (older pattern, still used):

```typescript
// ❌ Without type checking (JSX)
function MemoEditor({ content, onSave }) {
  return <textarea>{content}</textarea>;
}

// ✅ With TypeScript
interface MemoEditorProps {
  content: string;
  onSave: (content: string) => void;
}

const MemoEditor: React.FC<MemoEditorProps> = ({ content, onSave }) => {
  return <textarea>{content}</textarea>;
};
```

**Modern pattern** - Direct function declaration (preferred):

```typescript
// Modern approach - no React.FC needed
interface MemoEditorProps {
  content: string;
  onSave: (content: string) => void;
}

function MemoEditor({ content, onSave }: MemoEditorProps) {
  return <textarea>{content}</textarea>;
}
```

**Real example from Memos** (`web/src/contexts/AuthContext.tsx:26-27`):

```typescript
// Using children prop pattern
export function AuthProvider({ children }: { children: ReactNode }) {
  // ReactNode is the type for anything that can be rendered:
  // - strings, numbers, null, undefined, JSX, arrays of these
  const [state, setState] = useState<AuthState>({ /* ... */ });

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
```

### 2.2 Props Interface Pattern

```typescript
// Define props as an interface
interface ButtonProps {
  label: string;           // Required
  onClick: () => void;     // Function that returns nothing
  disabled?: boolean;      // Optional
  variant?: "primary" | "secondary";  // Union type
}

// Use destructuring with default values
function Button({ label, onClick, disabled = false, variant = "primary" }: ButtonProps) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={variant}
    >
      {label}
    </button>
  );
}
```

**Real example** (`web/src/components/MemoEditor/state/types.ts:8-31`):

```typescript
export interface EditorState {
  content: string;
  metadata: {
    visibility: Visibility;           // Enum from proto
    attachments: Attachment[];        // Array of type
    relations: MemoRelation[];        // Another array
    location?: Location;              // Optional
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
```

---

## 3. TypeScript Utility Types & Keywords

### 3.1 `Partial<T>` - Make all properties optional

**Concept**: Takes a type and makes every property optional.

```typescript
interface User {
  id: number;
  name: string;
  email: string;
}

// All properties are required
function updateUser(user: User) {
  // Must provide id, name, AND email
}

// All properties are optional
function partialUpdate(user: Partial<User>) {
  // Can provide any combination of properties
}

partialUpdate({ name: "John" });           // ✅ Valid
partialUpdate({ email: "john@example.com" }); // ✅ Valid
partialUpdate({}); // ✅ Valid (empty object)
```

**Real example from Memos** (`web/src/hooks/useMemoQueries.ts:84`):

```typescript
export function useUpdateMemo() {
  return useMutation({
    mutationFn: async ({ update, updateMask }: { update: Partial<Memo>; updateMask: string[] }) => {
      //          ^^^^^^^^^^^^^^^^^^ Only provide fields to update
      const memo = await memoServiceClient.updateMemo({
        memo: create(MemoSchema, update as Record<string, unknown>),
        updateMask: create(FieldMaskSchema, { paths: updateMask }),
      });
      return memo;
    },
  });
}

// Usage: Only update content, not the entire Memo
const { mutate: updateMemo } = useUpdateMemo();
updateMemo({
  update: { content: "New content" }, // Only sending content
  updateMask: ["content"]
});
```

### 3.2 `Required<T>` - Make all properties required

Opposite of `Partial`:

```typescript
interface User {
  id: number;
  name?: string;  // Optional
  email?: string; // Optional
}

type CompleteUser = Required<User>;
// Now equivalent to:
// { id: number; name: string; email: string; }
```

### 3.3 `Pick<T, K>` - Select specific properties

```typescript
interface User {
  id: number;
  name: string;
  email: string;
  password: string;
}

// Only pick id and name
type UserSummary = Pick<User, "id" | "name">;
// Equivalent to: { id: number; name: string; }

function displayUser(user: UserSummary) {
  console.log(`${user.id}: ${user.name}`);
  // No access to email or password
}
```

### 3.4 `Omit<T, K>` - Remove specific properties

```typescript
interface User {
  id: number;
  name: string;
  password: string; // Sensitive!
}

type SafeUser = Omit<User, "password">;
// Equivalent to: { id: number; name: string; }

function getUsers(): SafeUser[] {
  // Returns users without password field
}
```

### 3.5 `Record<K, V>` - Object with specific key/value types

```typescript
// Object with string keys and number values
type TagCounts = Record<string, number>;

const counts: TagCounts = {
  "react": 42,
  "typescript": 38,
  "go": 15
};
```

**Real example from Memos** (`web/src/hooks/useUserQueries.ts:106-114`):

```typescript
// Record<string, number> - string keys (tag names), number values (counts)
const tagCount: Record<string, number> = {};
for (const userStats of stats) {
  if (userStats.tagCount) {
    for (const [tag, count] of Object.entries(userStats.tagCount)) {
      tagCount[tag] = (tagCount[tag] || 0) + count;
    }
  }
}
return tagCount;
```

### 3.6 Union Types - `|` (OR operator)

```typescript
// Value can be string OR number
type ID = string | number;

// Function accepts multiple types
function display(value: string | number) {
  if (typeof value === "string") {
    console.log(value.toUpperCase()); // TypeScript knows it's a string here
  } else {
    console.log(value.toFixed(2));    // TypeScript knows it's a number here
  }
}
```

**Real example from Memos** (`web/src/components/MemoEditor/state/types.ts:33-49`):

```typescript
export type EditorAction =
  | { type: "INIT_MEMO"; payload: { content: string; /* ... */ } }
  | { type: "UPDATE_CONTENT"; payload: string }
  | { type: "SET_METADATA"; payload: Partial<EditorState["metadata"]> }
  | { type: "TOGGLE_FOCUS_MODE" }
  | { type: "RESET" };

// Discriminated union - the "type" field determines the shape
function reducer(state: EditorState, action: EditorAction) {
  switch (action.type) {
    case "UPDATE_CONTENT":
      // TypeScript knows action.payload is string here
      return { ...state, content: action.payload };
    case "TOGGLE_FOCUS_MODE":
      // TypeScript knows there's no payload here
      return { ...state, ui: { ...state.ui, isFocusMode: !state.ui.isFocusMode } };
  }
}
```

### 3.7 Intersection Types - `&` (AND operator)

```typescript
interface Name {
  name: string;
}

interface Age {
  age: number;
}

// Combine two types
type Person = Name & Age;
// Equivalent to: { name: string; age: number; }

const person: Person = {
  name: "John",
  age: 30
};
```

**Real example from Memos** (`web/src/types/proto/api/v1/memo_service_pb.ts:79`):

```typescript
export type Memo = Message<"memos.api.v1.Memo"> & {
  //      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ Intersection
  // Message base type AND custom properties
  name: string;
  state: State;
  // ... other properties
};
```

### 3.8 Generic Types - `<T>`

Generics allow types to be parameterized:

```typescript
// Generic function - works with any array type
function first<T>(arr: T[]): T | undefined {
  return arr[0];
}

// TypeScript infers T based on usage
first([1, 2, 3]);        // T is number
first(["a", "b", "c"]);  // T is string
first([true, false]);    // T is boolean
```

**Real example from Memos** (`web/src/hooks/useMemoQueries.ts:91-107`):

```typescript
// TypeScript generic - TData is the type of data in cache
onMutate: async ({ update }) => {
  // Cancel outgoing refetches
  await queryClient.cancelQueries({ queryKey: memoKeys.detail(update.name) });

  // <Memo> - generic type parameter
  const previousMemo = queryClient.getQueryData<Memo>(memoKeys.detail(update.name));

  if (previousMemo) {
    queryClient.setQueryData(memoKeys.detail(update.name), { ...previousMemo, ...update });
  }

  return { previousMemo };
},
```

### 3.9 `as const` - Create readonly literal types

```typescript
// Regular array - mutable, type is string[]
const fruits = ["apple", "banana", "orange"];
fruits.push("grape"); // Works
fruits[0] = "pear";   // Works

// as const - readonly, type is readonly ["apple", "banana", "orange"]
const fruits = ["apple", "banana", "orange"] as const;
fruits.push("grape"); // ❌ Error
fruits[0] = "pear";   // ❌ Error
```

**Real example from Memos** (`web/src/hooks/useMemoQueries.ts:10-17`):

```typescript
// Query keys factory with as const for type safety
export const memoKeys = {
  all: ["memos"] as const,
  lists: () => [...memoKeys.all, "list"] as const,
  list: (filters: Partial<ListMemosRequest>) => [...memoKeys.lists(), filters] as const,
  //                                      ^^^^^^ Creates readonly tuple type
  details: () => [...memoKeys.all, "detail"] as const,
  detail: (name: string) => [...memoKeys.details(), name] as const,
};

// Usage with strict type checking
useQuery({
  queryKey: memoKeys.list({ filter: "tag:react" }),
  // TypeScript knows exact type: ["memos", "list", Partial<ListMemosRequest>]
});
```

### 3.10 `keyof` - Get keys of an object type

```typescript
interface User {
  id: number;
  name: string;
  email: string;
}

type UserKeys = keyof User;
// Equivalent to: "id" | "name" | "email"

function getProperty(obj: User, key: keyof User) {
  return obj[key]; // Type-safe access
}

getProperty(user, "name");  // ✅ Valid - returns string
getProperty(user, "age");   // ❌ Error - "age" is not a key of User
```

### 3.11 `typeof` - Get type from a value

```typescript
const config = {
  apiUrl: "https://api.example.com",
  timeout: 5000,
  retries: 3
};

// Create a type from the value
type Config = typeof config;
// Equivalent to: { apiUrl: string; timeout: number; retries: number; }

function validateConfig(cfg: Config) {
  // Type-safe config validation
}
```

### 3.12 `Readonly<T>` - Make properties readonly

```typescript
interface User {
  name: string;
  email: string;
}

type ReadonlyUser = Readonly<User>;
// Equivalent to: { readonly name: string; readonly email: string; }

const user: ReadonlyUser = { name: "John", email: "john@example.com" };
user.name = "Jane"; // ❌ Error - cannot assign to readonly property
```

### 3.13 Enum vs Union Types

```typescript
// Enum - Runtime value
enum Visibility {
  PRIVATE = "PRIVATE",
  PROTECTED = "PROTECTED",
  PUBLIC = "PUBLIC"
}

// Union type - Compile-time only
type Visibility = "PRIVATE" | "PROTECTED" | "PUBLIC";
```

**Real example from Memos** (`web/src/types/proto/api/v1/memo_service_pb.ts:959-979`):

```typescript
// Generated enum from Protocol Buffer
export enum Visibility {
  VISIBILITY_UNSPECIFIED = 0,
  PRIVATE = 1,
  PROTECTED = 2,
  PUBLIC = 3,
}

// Usage in component
const visibility: Visibility = Visibility.PRIVATE;
```

---

## 4. React Query Deep Dive

### 4.1 What is React Query?

React Query (now TanStack Query) is a **data fetching and state management library**. Unlike Axios, it doesn't just make HTTP requests - it manages the entire lifecycle of server state.

**Key differences from Axios:**

| Aspect | Axios | React Query |
|--------|-------|-------------|
| What it does | HTTP client | Data fetching + caching + sync |
| Manual state management | Required | Built-in |
| Caching | Manual | Automatic |
| Background refetching | Manual | Automatic |
| Loading states | Manual `useState` | Built-in `isLoading` |
| Error states | Manual `try/catch` | Built-in `error` |
| Deduplication | Manual | Automatic |

### 4.2 Axios vs React Query: Side by Side

**Using Axios** (what you're used to):

```typescript
// ❌ Axios approach - lots of manual state management
function UserList() {
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function fetchUsers() {
      setIsLoading(true);
      try {
        const response = await axios.get("/api/users");
        setUsers(response.data);
      } catch (err) {
        setError(err as Error);
      } finally {
        setIsLoading(false);
      }
    }

    fetchUsers();
  }, []);

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return <ul>{users.map(user => <li key={user.id}>{user.name}</li>)}</ul>;
}
```

**Using React Query** (Memos pattern):

```typescript
// ✅ React Query approach - much cleaner
function UserList() {
  // One hook does everything!
  const { data: users, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: async () => {
      const response = await fetch("/api/users").then(r => r.json());
      return response;
    },
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return <ul>{users?.map(user => <li key={user.id}>{user.name}</li>)}</ul>;
}
```

### 4.3 Query Keys - The Foundation of Caching

**Query keys** uniquely identify cached data:

```typescript
// Simple key
useQuery({ queryKey: ["users"], queryFn: fetchUsers });

// Compound key (with parameters)
useQuery({
  queryKey: ["users", { status: "active", page: 1 }],
  queryFn: () => fetchUsers({ status: "active", page: 1 })
});

// Factory pattern (Memos approach)
export const userKeys = {
  all: ["users"] as const,
  details: () => [...userKeys.all, "detail"] as const,
  detail: (name: string) => [...userKeys.details(), name] as const,
  stats: () => [...userKeys.all, "stats"] as const,
  currentUser: () => [...userKeys.all, "current"] as const,
};

// Usage
useQuery({
  queryKey: userKeys.detail("users/123"),
  queryFn: () => fetchUser("users/123")
});
```

**Real example from Memos** (`web/src/hooks/useMemoQueries.ts:10-17`):

```typescript
// Hierarchical key factory for type safety
export const memoKeys = {
  all: ["memos"] as const,                              // Base key
  lists: () => [...memoKeys.all, "list"] as const,      // Derived: ["memos", "list"]
  list: (filters: Partial<ListMemosRequest>) =>        // Derived: ["memos", "list", filters]
    [...memoKeys.lists(), filters] as const,
  details: () => [...memoKeys.all, "detail"] as const,  // Derived: ["memos", "detail"]
  detail: (name: string) =>                             // Derived: ["memos", "detail", name]
    [...memoKeys.details(), name] as const,
};
```

### 4.4 Queries vs Mutations

**Query** - Fetching data (GET requests):

```typescript
const { data, isLoading, error } = useQuery({
  queryKey: ["memos", { filter: "tag:react" }],
  queryFn: async () => {
    const response = await memoServiceClient.listMemos({ filter: "tag:react" });
    return response;
  },
  staleTime: 1000 * 60,  // Cache for 1 minute
});
```

**Mutation** - Modifying data (POST/PUT/DELETE):

```typescript
const createMemo = useMutation({
  mutationFn: async (memoToCreate: Memo) => {
    const memo = await memoServiceClient.createMemo({ memo: memoToCreate });
    return memo;
  },
  onSuccess: (newMemo) => {
    // Invalidate queries to trigger refetch
    queryClient.invalidateQueries({ queryKey: memoKeys.lists() });
    // Or directly update cache
    queryClient.setQueryData(memoKeys.detail(newMemo.name), newMemo);
  },
});

// Usage
createMemo.mutate({ content: "Hello world", visibility: Visibility.PRIVATE });
```

**Real example from Memos** (`web/src/hooks/useMemoQueries.ts:61-78`):

```typescript
export function useCreateMemo() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (memoToCreate: Memo) => {
      const memo = await memoServiceClient.createMemo({ memo: memoToCreate });
      return memo;
    },
    onSuccess: (newMemo) => {
      // Strategy 1: Invalidate lists (triggers refetch)
      queryClient.invalidateQueries({ queryKey: memoKeys.lists() });

      // Strategy 2: Direct cache update (instant UI update)
      queryClient.setQueryData(memoKeys.detail(newMemo.name), newMemo);

      // Strategy 3: Invalidate related data
      queryClient.invalidateQueries({ queryKey: userKeys.stats() });
    },
  });
}
```

### 4.5 Optimistic Updates

Update UI immediately, rollback on error:

```typescript
export function useUpdateMemo() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ update, updateMask }: { update: Partial<Memo>; updateMask: string[] }) => {
      const memo = await memoServiceClient.updateMemo({
        memo: create(MemoSchema, update as Record<string, unknown>),
        updateMask: create(FieldMaskSchema, { paths: updateMask }),
      });
      return memo;
    },

    // BEFORE request - prepare optimistic update
    onMutate: async ({ update }) => {
      // Cancel outgoing refetches (prevent race conditions)
      await queryClient.cancelQueries({ queryKey: memoKeys.detail(update.name) });

      // Snapshot previous value for rollback
      const previousMemo = queryClient.getQueryData<Memo>(memoKeys.detail(update.name));

      // Optimistically update cache
      if (previousMemo) {
        queryClient.setQueryData(memoKeys.detail(update.name), {
          ...previousMemo,
          ...update
        });
      }

      return { previousMemo }; // Context for onError
    },

    // ON error - rollback
    onError: (_err, { update }, context) => {
      if (context?.previousMemo && update.name) {
        queryClient.setQueryData(memoKeys.detail(update.name), context.previousMemo);
      }
    },

    // ON success - update with server response
    onSuccess: (updatedMemo) => {
      queryClient.setQueryData(memoKeys.detail(updatedMemo.name), updatedMemo);
      queryClient.invalidateQueries({ queryKey: memoKeys.lists() });
    },
  });
}
```

### 4.6 React Query Configuration

**Real example from Memos** (`web/src/lib/query-client.ts:3-18`):

```typescript
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Data is "fresh" for 30 seconds - no refetch
      staleTime: 1000 * 30,

      // Keep in cache for 5 minutes after unused
      gcTime: 1000 * 60 * 5,  // Formerly cacheTime

      // Retry failed requests once
      retry: 1,

      // Refetch when user returns to tab
      refetchOnWindowFocus: true,

      // Refetch when network reconnects
      refetchOnReconnect: true,
    },
    mutations: {
      retry: 1,
    },
  },
});
```

### 4.7 Infinite Queries (Pagination)

**Real example from Memos** (`web/src/hooks/useMemoQueries.ts:29-47`):

```typescript
export function useInfiniteMemos(request: Partial<ListMemosRequest> = {}) {
  return useInfiniteQuery({
    queryKey: memoKeys.list(request),

    // pageParam is passed automatically by React Query
    queryFn: async ({ pageParam }) => {
      const response = await memoServiceClient.listMemos({
        ...request,
        pageToken: pageParam || "",  // Use pageToken for pagination
      });
      return response;
    },

    // Initial page parameter
    initialPageParam: "",

    // Extract next page token from response
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,

    // Cache settings
    staleTime: 1000 * 60,
    gcTime: 1000 * 60 * 5,
  });
}

// Usage in component
function MemoList() {
  const {
    data,
    isLoading,
    hasNextPage,
    fetchNextPage
  } = useInfiniteMemos({ filter: "tag:react" });

  // data.pages is an array of page results
  const memos = data?.pages.flatMap(page => page.memos) ?? [];

  return (
    <>
      {memos.map(memo => <MemoCard key={memo.name} memo={memo} />)}
      {hasNextPage && <button onClick={() => fetchNextPage()}>Load more</button>}
    </>
  );
}
```

---

## 5. Backend Stack: Go + gRPC vs Node/Express vs FastAPI

### 5.1 What is Go? (For JavaScript/Python Developers)

**Go (Golang)** is a programming language created by Google. Think of it as:

| Feature | JavaScript/Python | Go |
|---------|-------------------|-----|
| **Type system** | Dynamic (types checked at runtime) | Static (types checked at compile time) |
| **Execution** | Interpreted | Compiled to machine code |
| **Performance** | Slower | Much faster (closer to C/C++) |
| **Concurrency** | Event loop / async | Goroutines (lightweight threads) |
| **Learning curve** | Easier | A bit steeper |

**Why use Go for APIs?**
- **Performance**: 3-5x faster than Node.js for CPU-intensive tasks
- **Concurrency**: Can handle thousands of requests simultaneously with goroutines
- **Type safety**: Catches errors before code runs
- **Single binary**: Compiles to one executable file (no `node_modules`!)

**Hello World comparison:**

```javascript
// JavaScript/Node.js
function greet(name) {
  console.log("Hello, " + name);
}

greet("World");  // Works
greet(123);     // Also works (converts number to string)
greet.unknown;  // Crashes at runtime!
```

```go
// Go
package main

import "fmt"

func greet(name string) {
  fmt.Println("Hello, " + name)
}

func main() {
  greet("World")  // ✅ Works
  greet(123)     // ❌ Compile error! Can't pass int to string parameter
}
```

### 5.2 What is gRPC?

**gRPC** (Google Remote Procedure Call) is a framework for calling functions on a different machine as if they were local functions.

**Think of it like this:**

```typescript
// WITHOUT gRPC (REST API - what you're used to)
// Frontend:
const response = await fetch('/api/users/123');
const user = await response.json();
console.log(user.name);

// Backend (Express):
app.get('/api/users/:id', async (req, res) => {
  const user = await db.findUser(req.params.id);
  res.json(user);
});
```

```typescript
// WITH gRPC
// Frontend:
const user = await userService.getUser({ name: "users/123" });
console.log(user.name);
// ^ Looks like a local function call!

// Backend (gRPC):
func (s *UserService) GetUser(ctx context.Context, request *GetUserRequest) (*User, error) {
  user := db.FindUser(request.Name)
  return user, nil
}
```

**Key differences:**
- **REST**: You define URLs and HTTP methods manually
- **gRPC**: You define functions in a `.proto` file, then call them directly

### 5.3 What are Protocol Buffers?

**Protocol Buffers (protobuf)** is a way to define data structures and API contracts. Think of it as:

> "TypeScript, but for defining APIs instead of just code"

**JSON vs Protocol Buffers:**

```json
// JSON - text-based, human-readable
{
  "name": "John Doe",
  "age": 30,
  "email": "john@example.com"
}
```

```protobuf
// Protocol Buffers - binary, compact, fast
message User {
  string name = 1;
  int32 age = 2;
  string email = 3;
}
// When encoded to binary: 0A084A6F686E20446F65101E1A0E6A6F686E406578616D706C652E636F6D
// (Much smaller and faster to parse!)
```

**Why Protocol Buffers?**

| Aspect | JSON | Protocol Buffers |
|--------|------|------------------|
| **Format** | Text (human-readable) | Binary (compact) |
| **Size** | Larger (lots of `{}` `""` `:`) | 2-5x smaller |
| **Speed** | Slower to parse | Much faster |
| **Schema** | None (or OpenAPI separately) | Built-in (.proto file) |
| **Code generation** | Manual | Automatic |

### 5.4 How It All Works Together

**The complete flow:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DEVELOPMENT TIME                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. Developer writes .proto file:                                       │
│                                                                         │
│     service MemoService {                                               │
│       rpc CreateMemo(CreateMemoRequest) returns (Memo);                 │
│     }                                                                  │
│                                                                         │
│     message Memo {                                                      │
│       string name = 1;                                                  │
│       string content = 2;                                               │
│     }                                                                  │
│                                                                         │
│  2. Run: buf generate                                                  │
│                                                                         │
│  3. Get auto-generated code:                                           │
│     → TypeScript types (frontend)                                      │
│     → Go types and service interface (backend)                         │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                         RUNTIME (when app runs)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  FRONTEND (TypeScript):                                                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  const response = await memoServiceClient.createMemo({           │    │
│  │    memo: { name: "memos/123", content: "Hello" }                │    │
│  │  });                                                             │    │
│  │                                                                   │    │
│  │  // Internally:                                                   │    │
│  │  // 1. Validates input against TypeScript types                  │    │
│  │  // 2. Serializes to binary protobuf format                      │    │
│  │  // 3. Sends via HTTP POST                                       │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                           │                                              │
│                    [Binary protobuf over HTTP]                           │
│                           │                                              │
│  BACKEND (Go):                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  func (s *MemoService) CreateMemo(ctx, request) (*Memo, error) { │    │
│  │    // 1. Receives binary protobuf                                │    │
│  │    // 2. Deserializes to Go struct                               │    │
│  │    // 3. Type-safe access to fields                             │    │
│  │    memo := &store.Memo{                                          │    │
│  │      Name: request.Name,      // Go knows this is string         │    │
│  │      Content: request.Content, // Go knows this is string        │    │
│  │    }                                                             │    │
│  │    // 4. Save to database                                        │    │
│  │    // 5. Return response (auto-serialized to protobuf)          │    │
│  │  }                                                               │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.5 The .proto File: Your API Contract

The `.proto` file is the **single source of truth** for your API. Let's break it down:

```protobuf
// proto/api/v1/memo_service.proto

// ===== SYNTAX =====
// "proto3" is the version of protobuf we're using (like ES6 vs ES5 in JS)
syntax = "proto3";

// ===== PACKAGE =====
// Like a namespace in TypeScript or package in Java
// Prevents naming conflicts when you have multiple services
package memos.api.v1;

// ===== IMPORTS =====
// Import common types from other protobuf files
import "google/protobuf/timestamp.proto";
import "google/api/annotations.proto";

// ===== ENUMS =====
// Like TypeScript enums - defines a set of possible values
enum Visibility {
  VISIBILITY_UNSPECIFIED = 0;  // = 0 is required for default values in proto3
  PRIVATE = 1;
  PROTECTED = 2;
  PUBLIC = 3;
}

enum State {
  STATE_NORMAL = 0;
  STATE_ARCHIVED = 1;
}

// ===== MESSAGES =====
// Like TypeScript interfaces - defines data structures
// The numbers (1, 2, 3...) are field identifiers, used in binary encoding

message Memo {
  // string name = 1;
  // ^^^^  ^^^^  ^
  //  |    |    |
  //  |    |    └── Field number (unique within this message)
  //  |    └──────── Field name
  //  └───────────── Field type (string, int32, bool, etc.)

  string name = 1;                            // Required - must have a value
  State state = 2;                             // Enum type
  string creator = 3;
  google.protobuf.Timestamp create_time = 4;   // Import from google protobuf
  string content = 7;                          // Notice we skipped 5,6 (that's OK!)
  Visibility visibility = 9;
  repeated string tags = 10;                   // repeated = array in TypeScript
  bool pinned = 11;                            // bool = boolean in TypeScript
}

// ===== SERVICE DEFINITION =====
// Defines the API - what functions can be called
service MemoService {
  // rpc = Remote Procedure Call (like a function)
  // CreateMemo is the function name
  // CreateMemoRequest is what you send
  // Memo is what you get back

  rpc CreateMemo(CreateMemoRequest) returns (Memo);
  rpc ListMemos(ListMemosRequest) returns (ListMemosResponse);
  rpc GetMemo(GetMemoRequest) returns (Memo);
  rpc UpdateMemo(UpdateMemoRequest) returns (Memo);
  rpc DeleteMemo(DeleteMemoRequest) returns (google.protobuf.Empty);
}

// ===== REQUEST/RESPONSE MESSAGES =====
message CreateMemoRequest {
  Memo memo = 1;         // The memo to create
  string memo_id = 2;    // Optional: specify your own ID
}

message ListMemosRequest {
  int32 page_size = 1;       // int32 = number (32-bit integer)
  string page_token = 2;     // For pagination
  State state = 3;           // Filter by state
  string filter = 5;         // CEL expression for filtering
}

message ListMemosResponse {
  repeated Memo memos = 1;           // Array of memos
  string next_page_token = 2;        // Token for next page
}
```

**Important notes about field numbers:**
- Each field must have a **unique number** (1, 2, 3, etc.)
- These numbers are used in the **binary encoding**, not the field names
- **Never change** a field number once you've shipped your API
- You can skip numbers (like we skipped 5, 6) - this is actually recommended for forward compatibility
- Numbers 1-15 use **1 byte** in encoding
- Numbers 16-2047 use **2 bytes**
- Reserve low numbers for frequently-used fields

### 5.6 From .proto to Generated Code

When you run `buf generate`, it creates code in **multiple languages** from the **same .proto file**:

**Generated TypeScript** (for frontend):
```typescript
// web/src/types/proto/api/v1/memo_service_pb.ts
// Auto-generated - DO NOT EDIT

// TypeScript type for Memo
export type Memo = Message<"memos.api.v1.Memo"> & {
  name: string;
  state: State;
  creator: string;
  createTime?: Timestamp;
  content: string;
  visibility: Visibility;
  tags: string[];
  pinned: boolean;
};

// TypeScript enum for Visibility
export enum Visibility {
  VISIBILITY_UNSPECIFIED = 0,
  PRIVATE = 1,
  PROTECTED = 2,
  PUBLIC = 3,
}

// TypeScript type for requests
export type CreateMemoRequest = Message<"memos.api.v1.CreateMemoRequest"> & {
  memo?: Memo;
  memoId: string;
};
```

**Generated Go** (for backend):
```go
// proto/gen/api/v1/memo_service.pb.go
// Auto-generated - DO NOT EDIT

// Go struct for Memo
type Memo struct {
    Name       string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
    State      State                  `protobuf:"varint,2,opt,name=state,proto3,enum=memos.api.v1.State" json:"state,omitempty"`
    Creator    string                 `protobuf:"bytes,3,opt,name=creator,proto3" json:"creator,omitempty"`
    Content    string                 `protobuf:"bytes,7,opt,name=content,proto3" json:"content,omitempty"`
    Visibility Visibility             `protobuf:"varint,9,opt,name=visibility,proto3,enum=memos.api.v1.Visibility" json:"visibility,omitempty"`
    Tags       []string               `protobuf:"bytes,10,rep,name=tags,proto3" json:"tags,omitempty"`
    Pinned     bool                   `protobuf:"varint,11,opt,name=pinned,proto3" json:"pinned,omitempty"`
}

// Go enum for Visibility
type Visibility int32
const (
    Visibility_VISIBILITY_UNSPECIFIED Visibility = 0
    Visibility_PRIVATE                 Visibility = 1
    Visibility_PROTECTED               Visibility = 2
    Visibility_PUBLIC                  Visibility = 3
)
```

**The magic:** Both languages have **matching types** generated from the **same source**!

### 5.7 Architecture Overview: Comparing Stacks

**Node/Express** (what you're used to):
```
Client (browser)
    ↓
  JSON payload
    ↓
Express Route (/api/users)
    ↓
Controller (validates manually)
    ↓
Service (business logic)
    ↓
Repository/Model
    ↓
Database
```

**FastAPI** (Python):
```
Client (browser)
    ↓
  JSON payload
    ↓
FastAPI Route (/users)
    ↓
Pydantic validates automatically
    ↓
Service (business logic)
    ↓
Repository
    ↓
Database
```

**Memos (Go + gRPC + Connect)**:
```
Client (browser)
    ↓
  TypeScript types (from .proto)
    ↓
  Binary protobuf payload
    ↓
Connect RPC (HTTP-based RPC)
    ↓
Interceptors (auth, logging)
    ↓
gRPC Service (auto-generated interface)
    ↓
Service implementation (you write this)
    ↓
Store (data access layer)
    ↓
Database
```

### 5.8 Why Go + gRPC Instead of Node/Express?

| Feature | Node/Express | Go + gRPC | Why it matters |
|---------|-------------|-----------|----------------|
| **Protocol** | JSON (text) | Protocol Buffers (binary) | Binary is 2-5x smaller and faster |
| **Validation** | Manual (or Joi/Zod) | Automatic from .proto | Less code, fewer bugs |
| **Type Safety** | TypeScript (compile-time) | Go (compile-time + generated) | Catches more errors before runtime |
| **Code Generation** | Write types manually | Auto-generated from .proto | Change .proto → regenerate → types update everywhere |
| **Payload Size** | Larger JSON | Smaller binary | Less bandwidth usage |
| **Performance** | Single-threaded event loop | Multi-threaded goroutines | Better for concurrent requests |
| **API Documentation** | Swagger/OpenAPI (separate) | .proto file (single source) | Docs never out of sync |
| **Cross-language** | JSON works everywhere | Proto generates for any language | Frontend can be TS, backend Go, mobile Java |

### 5.9 Detailed Request Flow

Let's trace a request from frontend to backend:

**Step 1: Frontend makes request**
```typescript
// web/src/hooks/useMemoQueries.ts
export function useCreateMemo() {
  return useMutation({
    mutationFn: async (memoToCreate: Memo) => {
      // memoToCreate is type-checked against Memo type from .proto
      const memo = await memoServiceClient.createMemo({
        memo: memoToCreate
      });
      return memo;
    },
  });
}

// Usage in component:
const { mutate: createMemo } = useCreateMemo();
createMemo({
  name: "memos/123",
  content: "Hello world",  // TypeScript ensures this is string
  visibility: Visibility.PRIVATE,  // TypeScript ensures this is valid enum
  // tags: 123,  // ❌ Would be compile error!
});
```

**Step 2: Connect RPC client serializes request**
```typescript
// Inside @bufbuild/connect (the library)
// The client does this automatically:

// 1. Validate the input matches the .proto schema
// 2. Serialize to binary protobuf format
const binaryData = serialize(CreateMemoRequestSchema, {
  memo: {
    name: "memos/123",
    content: "Hello world",
    visibility: Visibility.PRIVATE,
  }
});

// 3. Send as HTTP POST
fetch("http://localhost:8081/memos.api.v1.MemoService/CreateMemo", {
  method: "POST",
  body: binaryData,  // Binary, not JSON!
  headers: {
    "Content-Type": "application/connect+proto",
  }
});
```

**Step 3: Backend receives and deserializes**
```go
// server/router/api/v1/memo_service.go

// The interceptor chain runs first:
// 1. Metadata interceptor - adds auth token to context
// 2. Logging interceptor - logs request
// 3. Recovery interceptor - handles panics
// 4. Auth interceptor - validates JWT

// Then your service function is called:
func (s *APIV1Service) CreateMemo(ctx context.Context, request *v1.CreateMemoRequest) (*v1.Memo, error) {
  // request is already deserialized from binary to Go struct
  // Go knows the types because they were generated from .proto

  // Get user ID from context (set by auth interceptor)
  userID := ctx.Value("user_id").(string)

  // Validate request (proto already did basic validation)
  if request.Memo == nil {
    return nil, status.Error(codes.InvalidArgument, "memo is required")
  }

  // Convert proto type to internal store type
  memo := &store.Memo{
    CreatorID:  userID,
    Content:    request.Memo.Content,    // string field
    Visibility: store.Visibility(request.Memo.Visibility),  // enum field
  }

  // Call store layer to save to database
  createdMemo, err := s.Store.CreateMemo(ctx, memo)
  if err != nil {
    return nil, status.Error(codes.Internal, "failed to create memo")
  }

  // Convert back to proto type (return value is auto-serialized)
  return convertToProtoMemo(createdMemo), nil
}
```

**Step 4: Response serialized and sent back**
```go
// The Connect framework does this automatically:

// 1. Take the Go struct you returned
// 2. Serialize to binary protobuf format
// 3. Set HTTP response headers
// 4. Send response body
```

**Step 5: Frontend receives and deserializes**
```typescript
// Inside @bufbuild/connect (the library)

// 1. Receive binary response
// 2. Deserialize to TypeScript object
// 3. Validate against Memo type
// 4. Return typed Memo object

const memo: Memo = response;
// ^ TypeScript knows this has name, content, visibility, etc.
```

### 5.10 Comparing API Implementations Side-by-Side

Let's compare the **same endpoint** across three stacks:

**Node/Express** (what you know):
```typescript
// routes/memos.ts
import express from 'express';
const router = express.Router();

// ❌ Manual validation
router.get('/memos', async (req, res) => {
  const { pageSize, pageToken } = req.query;

  // Must manually validate
  if (typeof pageSize !== 'string' || isNaN(Number(pageSize))) {
    return res.status(400).json({ error: 'Invalid pageSize' });
  }

  // Must manually parse query params
  const size = Number(pageSize) || 50;

  // Must manually map database results to response shape
  const memos = await db.memos.findMany({
    take: size,
    cursor: pageToken ? { id: pageToken } : undefined
  });

  // Must manually format response
  res.json({
    memos: memos.map(m => ({
      id: m.id,
      content: m.content,
      createdAt: m.createdAt.toISOString()
    })),
    nextPageToken: memos[memos.length - 1]?.id
  });
});
```

**FastAPI** (Python):
```python
# routers/memos.py
from fastapi import APIRouter, Query
from pydantic import BaseModel
from typing import List, Optional

router = APIRouter()

# ✅ Pydantic provides validation
class Memo(BaseModel):
    id: str
    content: str
    created_at: datetime

class ListMemosResponse(BaseModel):
    memos: List[Memo]
    next_page_token: Optional[str]

# ✅ Pydantic validates query params
@router.get("/memos", response_model=ListMemosResponse)
async def list_memos(
    page_size: int = Query(50, le=1000),
    page_token: Optional[str] = None
):
    # Pydantic validated page_size is int and <= 1000
    memos = await db.memos.find_many(
        take=page_size,
        cursor={"id": page_token} if page_token else None
    )

    # Pydantic validates response shape
    return ListMemosResponse(
        memos=[Memo(**m) for m in memos],
        next_page_token=memos[-1].id if memos else None
    )
```

**Memos (Go + gRPC)**:
```go
// server/router/api/v1/memo_service.go

// ✅ Types auto-generated from .proto
// ✅ Validation automatic from .proto schema
func (s *APIV1Service) ListMemos(ctx context.Context, request *v1.ListMemosRequest) (*v1.ListMemosResponse, error) {
    // ✅ No manual validation needed - proto did it
    // request.PageSize is int32 (guaranteed by proto)
    // request.PageToken is string (guaranteed by proto)

    // ✅ Type-safe field access
    memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
        PageSize:  int(request.PageSize),   // Go knows this is int32
        PageToken: request.PageToken,       // Go knows this is string
    })
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to list memos")
    }

    // ✅ Response type auto-generated
    return &v1.ListMemosResponse{
        Memos:        memos,    // Go converts automatically
        NextPageToken: nextPageToken,
    }, nil
}
```

**Key differences:**

| Aspect | Node/Express | FastAPI | Go + gRPC |
|--------|-------------|---------|-----------|
| **Validation** | Manual | Pydantic | Automatic (proto) |
| **Type safety** | TypeScript (optional) | Pydantic | Go + proto (strict) |
| **Request parsing** | Manual (`req.query`) | Automatic | Automatic |
| **Response format** | Manual JSON | Pydantic model | Auto-generated |
| **Code to maintain** | All of it | Less | Very little |
| **Single source of truth** | None | Pydantic models | .proto file |

### 5.11 What is Connect RPC?

**Connect RPC** is a modern RPC framework that works over HTTP/1.1 and HTTP/2. It's designed to be:

- **Compatible with browsers** (unlike traditional gRPC which needs HTTP/2)
- **Simple to use** from TypeScript/JavaScript
- **Compatible with gRPC** on the backend

**Think of Connect as:**
> "A bridge between your TypeScript frontend and Go backend that speaks gRPC"

**Why not just use gRPC directly?**

Traditional gRPC has problems in browsers:
- Requires HTTP/2 (not all browsers support it well)
- Binary protocol is hard to debug
- Requires special tooling

**Connect RPC solves this:**

```
┌─────────────────────────────────────────────────────────────┐
│                   CONNECT RPC ARCHITECTURE                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Frontend (Browser)          Backend (Go)                    │
│                                                              │
│  TypeScript types  ──────▶  Go types                         │
│  (from .proto)              (from .proto)                    │
│       │                            ▲                         │
│       │                            │                         │
│       ▼                            │                         │
│  Connect Client  ────────▶  Connect Server                   │
│  (@bufbuild/connect)       (connect-go)                      │
│       │                            │                         │
│       │    HTTP POST (binary)       │                         │
│       └────────────────────────────┘                         │
│                                                              │
│  Uses HTTP/1.1 or HTTP/2                                     │
│  Sends binary protobuf in request body                       │
│  Works in all browsers                                       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Frontend usage:**

```typescript
// web/src/connect.ts
import { createPromiseClient } from "@bufbuild/connect";
import { createConnectTransport } from "@bufbuild/connect-connect";
import { MemoService } from "@/types/proto/api/v1/memo_service_pb";

// Create client (done once, at app startup)
export const memoServiceClient = createPromiseClient(
  MemoService,  // Service definition from .proto
  createConnectTransport({
    baseUrl: "http://localhost:8081",  // Backend URL
  })
);

// Use in components
const { data } = useQuery({
  queryKey: ["memos"],
  queryFn: () => memoServiceClient.listMemos({ pageSize: 50 })
  //    ^^^^^^^^^^^^^^^
  // This looks like a local function call, but it makes an HTTP request!
});
```

**What happens behind the scenes:**

```typescript
// When you call memoServiceClient.listMemos({ pageSize: 50 }):
//
// 1. TypeScript validates that pageSize is a number
// 2. Connect serializes { pageSize: 50 } to binary protobuf
// 3. Makes HTTP POST to http://localhost:8081/memos.api.v1.MemoService/ListMemos
// 4. Request body: <binary protobuf data>
// 5. Response body: <binary protobuf data>
// 6. Deserializes response to TypeScript object
// 7. Returns typed ListMemosResponse
```

### 5.12 Dual Protocol: Connect RPC + REST

Memos exposes **two protocols** from the same Go backend:

**Why two protocols?**

| Use Case | Best Protocol | Why |
|----------|---------------|-----|
| Browser (React app) | Connect RPC | Type-safe, fast, smaller payloads |
| External tools (curl, Postman) | REST (gRPC-Gateway) | Familiar, debuggable, works everywhere |
| Mobile apps | Either | Connect is better, REST is simpler |
| Webhooks | REST | Standard HTTP, easier to integrate |

**Protocol 1: Connect RPC (for browser)**

```typescript
// Frontend uses Connect RPC
const response = await memoServiceClient.listMemos({ pageSize: 50 });
// ^ Makes POST to: http://localhost:8081/memos.api.v1.MemoService/ListMemos
// Content-Type: application/connect+proto
// Body: <binary protobuf>
```

**Protocol 2: REST/gRPC-Gateway (for external tools)**

```bash
# External tools use REST
curl "http://localhost:8081/api/v1/memos?page_size=50"
# ^ Makes GET to: /api/v1/memos
# Accept: application/json
# Response: <JSON>
```

**How both protocols work in Go:**

```go
// server/router/api/v1/v1.go

func (s *APIV1Service) RegisterGateway(mux *http.ServeMux) {
    // ===== Connect RPC (for browsers) =====
    // Registers: /memos.api.v1.MemoService/*
    connect.NewService(s.Handler(), mux)

    // ===== REST/gRPC-Gateway (for external tools) =====
    // Registers: /api/v1/memos
    // Generated from google.api.annotations in .proto
    // Example in .proto:
    //   rpc ListMemos(ListMemosRequest) returns (ListMemosResponse) {
    //     option (google.api.http) = {
    //       get: "/api/v1/memos"
    //     };
    //   }
}

// Both protocols call the same service function!
func (s *APIV1Service) ListMemos(ctx context.Context, request *v1.ListMemosRequest) (*v1.ListMemosResponse, error) {
    // This code is called by BOTH Connect RPC and REST!
    memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
        PageSize:  int(request.PageSize),
        PageToken: request.PageToken,
    })
    return &v1.ListMemosResponse{
        Memos:        memos,
        NextPageToken: nextPageToken,
    }, nil
}
```

**Request comparison:**

```
CONNECT RPC Request:
POST /memos.api.v1.MemoService/ListMemos
Content-Type: application/connect+proto

<binary protobuf data>

Response:
Content-Type: application/connect+proto
<binary protobuf data>
```

```
REST Request:
GET /api/v1/memos?page_size=50
Accept: application/json

Response:
Content-Type: application/json
{
  "memos": [...],
  "nextPageToken": "..."
}
```

### 5.13 Interceptors: Middleware for gRPC

**Interceptors** are like middleware in Express, but for gRPC/Connect. They run before and after your service functions.

**Request flow with interceptors:**

```
Client Request
    ↓
[Interceptor 1: Metadata]  - Adds auth token to context
    ↓
[Interceptor 2: Logging]   - Logs request details
    ↓
[Interceptor 3: Recovery]  - Catches panics
    ↓
[Interceptor 4: Auth]      - Validates JWT token
    ↓
Your Service Function       - Your business logic
    ↓
[Interceptor 4: Auth]      - Post-auth processing
    ↓
[Interceptor 3: Recovery]  - Error handling
    ↓
[Interceptor 2: Logging]   - Logs response
    ↓
[Interceptor 1: Metadata]  - Adds response headers
    ↓
Client Response
```

**Real example from Memos**:

```go
// server/router/api/v1/connect_interceptors.go

// 1. Metadata Interceptor - Adds context to requests
func NewMetadataInterceptor() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Add request ID for tracing
            requestID := uuid.New().String()
            ctx = context.WithValue(ctx, "request_id", requestID)

            // Call next interceptor/service
            return next(ctx, req)
        }
    }
}

// 2. Logging Interceptor - Logs all requests
func NewLoggingInterceptor() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            start := time.Now()

            // Log request
            log.Printf("Request: %s %s", req.Spec().Procedure, req.Peer())

            // Call next
            resp, err := next(ctx, req)

            // Log response
            duration := time.Since(start)
            log.Printf("Response: %s (took %s)", req.Spec().Procedure, duration)

            return resp, err
        }
    }
}

// 3. Recovery Interceptor - Catches panics
func NewRecoveryInterceptor() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("Panic recovered: %v", r)
                }
            }()

            return next(ctx, req)
        }
    }
}

// 4. Auth Interceptor - Validates JWT
func NewAuthInterceptor(authenticator *Authenticator) connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Skip auth for public endpoints
            if isPublicEndpoint(req.Spec().Procedure) {
                return next(ctx, req)
            }

            // Get token from headers
            token := extractToken(req.Header())

            // Validate token
            userID, err := authenticator.ValidateAccessToken(token)
            if err != nil {
                return nil, status.Error(codes.Unauthenticated, "invalid token")
            }

            // Add user ID to context
            ctx = context.WithValue(ctx, "user_id", userID)

            // Call next
            return next(ctx, req)
        }
    }
}

// Chain all interceptors together
func (s *APIV1Service) NewHandler() http.Handler {
    return connect.NewService(
        s.Handler(),
        connect.WithInterceptors(
            NewMetadataInterceptor(),    // Runs first
            NewLoggingInterceptor(),     // Runs second
            NewRecoveryInterceptor(),    // Runs third
            NewAuthInterceptor(s.auth),  // Runs fourth
        ),
    )
}
```

**Your service function gets the context from interceptors:**

```go
func (s *APIV1Service) CreateMemo(ctx context.Context, request *v1.CreateMemoRequest) (*v1.Memo, error) {
    // Get user ID that auth interceptor added
    userID := ctx.Value("user_id").(string)

    // Get request ID that metadata interceptor added
    requestID := ctx.Value("request_id").(string)

    // Business logic...
}
```

### 5.14 Store Layer: Data Access Pattern

**Store** is Memos' data access layer (like Repository pattern in other apps).

**Why separate Store from Service?**

| Layer | Responsibility |
|-------|----------------|
| **Service** (`memo_service.go`) | - Validate request <br> - Call store <br> - Convert types <br> - Return response |
| **Store** (`store/memo.go`) | - Database operations <br> - Caching <br> - Transactions <br> - SQL queries |

**Store interface (defines all data operations):**

```go
// store/driver.go
type Driver interface {
    // Memo operations
    CreateMemo(ctx context.Context, create *Memo) (*Memo, error)
    ListMemos(ctx context.Context, find *FindMemo) ([]*Memo, error)
    GetMemo(ctx context.Context, find *FindMemo) (*Memo, error)
    UpdateMemo(ctx context.Context, update *UpdateMemo) error
    DeleteMemo(ctx context.Context, delete *DeleteMemo) error

    // User operations
    CreateUser(ctx context.Context, create *User) (*User, error)
    ListUsers(ctx context.Context, find *FindUser) ([]*User, error)
    // ... etc
}
```

**Service uses Store:**

```go
// server/router/api/v1/memo_service.go
func (s *APIV1Service) CreateMemo(ctx context.Context, request *v1.CreateMemoRequest) (*v1.Memo, error) {
    // 1. Get user from context (set by auth interceptor)
    userID := ctx.Value("user_id").(string)

    // 2. Convert proto type to store type
    memo := &store.Memo{
        CreatorID:  userID,
        Content:    request.Memo.Content,
        Visibility: store.Visibility(request.Memo.Visibility),
        Tags:       request.Memo.Tags,
    }

    // 3. Call store (doesn't know about proto types)
    createdMemo, err := s.Store.CreateMemo(ctx, memo)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to create memo")
    }

    // 4. Convert store type back to proto type
    return convertToProtoMemo(createdMemo), nil
}
```

**Store implementation (SQLite example):**

```go
// store/db/sqlite/memo.go
func (d *Driver) CreateMemo(ctx context.Context, create *store.Memo) (*store.Memo, error) {
    // 1. Generate ID
    id := generateID("memos")

    // 2. Execute SQL insert
    _, err := d.db.ExecContext(ctx, `
        INSERT INTO memo (
            id, creator_id, content, visibility, tags, created_ts
        ) VALUES (?, ?, ?, ?, ?, ?)
    `, id, create.CreatorID, create.Content, create.Visibility,
       strings.Join(create.Tags, ","), time.Now())

    if err != nil {
        return nil, fmt.Errorf("failed to insert memo: %w", err)
    }

    // 3. Fetch and return created memo
    return d.GetMemo(ctx, &store.FindMemo{ID: &id})
}
```

**This separation enables:**
- **Multiple databases**: Same interface, different implementations (SQLite, MySQL, PostgreSQL)
- **Testing**: Mock store for unit tests
- **Caching**: Store wrapper adds caching layer
- **Transactions**: Store handles database transactions

### 5.15 Interceptors Explained (More Detail)

**Express middleware you're used to:**

```typescript
// Express middleware
app.use((req, res, next) => {
  console.log('Request:', req.method, req.url);
  next();  // Call next middleware/route
});

app.get('/memos', (req, res) => {
  res.json({ memos: [] });
});
```

**gRPC/Connect interceptors are similar but more powerful:**

```go
// Connect interceptor
func NewLoggingInterceptor() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Before request
            log.Printf(">>> %s %v", req.Spec().Procedure, req.Msg())

            // Call next interceptor/service
            resp, err := next(ctx, req)

            // After response
            log.Printf("<<< %s error=%v", req.Spec().Procedure, err)

            return resp, err
        }
    }
}
```

**Key differences:**

| Aspect | Express Middleware | gRPC/Connect Interceptor |
|--------|-------------------|-------------------------|
| **Works with** | HTTP request/response | Unary (single req/res) or streaming |
| **Error handling** | Try/catch or next(err) | Return error from function |
| **Context passing** | `req.context` | Go `context.Context` |
| **Can modify request** | Yes, `req.body = ...` | No (protobuf is immutable) |
| **Can modify response** | Yes, `res.json(...)` | No (protobuf is immutable) |
| **Access to method name** | `req.url` or `req.route` | `req.Spec().Procedure` |

### 5.16 Complete Request Flow with All Components

Let's trace a complete request from button click to UI update:

```
┌─────────────────────────────────────────────────────────────────────────┐
│  USER ACTION                                                            │
│  User clicks "Create Memo" button in browser                           │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  FRONTEND (TypeScript)                                                  │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ function MemoEditor() {                                         │    │
│  │   const { mutate: createMemo } = useCreateMemo();               │    │
│  │                                                                  │    │
│  │   return (                                                       │    │
│  │     <button onClick={() => createMemo({                         │    │
│  │       memo: {                                                    │    │
│  │         name: "memos/123",                                       │    │
│  │         content: "Hello world",                                  │    │
│  │         visibility: Visibility.PRIVATE,                          │    │
│  │       }                                                          │    │
│  │     })}>                                                         │    │
│  │     Create Memo                                                  │    │
│  │     </button>                                                    │    │
│  │   );                                                             │    │
│  │ }                                                                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  React Query Hook (useCreateMemo)                               │    │
│  │  - Validates input against TypeScript types                     │    │
│  │  - Calls memoServiceClient.createMemo()                         │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Connect RPC Client (@bufbuild/connect)                         │    │
│  │  1. Validates request matches .proto schema                     │    │
│  │  2. Serializes to binary protobuf                               │    │
│  │  3. Creates HTTP POST request                                   │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                    HTTP POST (binary protobuf)
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  BACKEND (Go)                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Interceptor Chain                                               │    │
│  │  1. Metadata: Adds request ID to context                        │    │
│  │  2. Logging: Logs "CreateMemo called"                           │    │
│  │  3. Recovery: Catches any panics                                │    │
│  │  4. Auth: Validates JWT, extracts user ID                       │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  gRPC Service Function                                           │    │
│  │  func (s *APIV1Service) CreateMemo(ctx, request) (*Memo, error) {│    │
│  │    // Already deserialized from binary to Go struct             │    │
│  │    userID := ctx.Value("user_id").(string)                       │    │
│  │                                                                   │    │
│  │    memo := &store.Memo{                                           │    │
│  │      CreatorID:  userID,                                         │    │
│  │      Content:    request.Memo.Content,                           │    │
│  │      Visibility: store.Visibility(request.Memo.Visibility),      │    │
│  │    }                                                              │    │
│  │                                                                   │    │
│  │    createdMemo, err := s.Store.CreateMemo(ctx, memo)            │    │
│  │    return convertToProtoMemo(createdMemo), nil                   │    │
│  │  }                                                                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Store Layer (Data Access)                                       │    │
│  │  func (d *Driver) CreateMemo(ctx, create *Memo) (*Memo, error) {│    │
│  │    // Executes SQL INSERT                                        │    │
│  │    db.Exec(`INSERT INTO memo (id, content, ...) VALUES (...)`)   │    │
│  │    // Returns created memo                                       │    │
│  │  }                                                                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Database (SQLite/MySQL/PostgreSQL)                              │    │
│  │  - Stores memo data                                              │    │
│  │  - Returns created row                                           │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  Go structs converted to binary protobuf                              │
│                    HTTP Response (binary protobuf)                    │
│                                  │                                       │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  FRONTEND (TypeScript)                                                  │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Connect RPC Client                                              │    │
│  │  - Receives binary response                                      │    │
│  │  - Deserializes to TypeScript object                             │    │
│  │  - Validates against Memo type                                   │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  React Query (onSuccess callback)                               │    │
│  │  - Invalidates queries (triggers refetch)                        │    │
│  │  - Updates cache with new memo                                   │    │
│  │  - React re-renders with new data                                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                  │                                       │
│                                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  UI Updates                                                       │    │
│  │  - New memo appears in list                                      │    │
│  │  - Success notification shown                                     │    │
│  │  - Form cleared                                                   │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.17 FromStore and ToStore Pattern: Type Conversion Functions

In the Memos codebase, you'll notice pairs of functions with names like:
- `convertInstanceMemoRelatedSettingFromStore()`
- `convertInstanceMemoRelatedSettingToStore()`
- `convertUserFromStore()`
- `convertUserToStore()`

**Why two type systems?**

The backend uses **two separate protobuf type packages**:

```
proto/
├── api/v1/         → v1pb (API Layer - exposed to clients)
└── store/          → storepb (Store Layer - internal storage)
```

**v1pb** (`proto/api/v1/instance_service.proto`):
- Public API types
- What clients (frontend, mobile apps) send and receive
- Stable, versioned interface

**storepb** (`proto/store/instance_setting.proto`):
- Internal storage types
- What the database layer uses
- Can change without breaking public API

**The conversion flow:**

```
┌─────────────────────────────────────────────────────────────────┐
│                         Request Flow                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Frontend sends: v1pb.InstanceSetting                           │
│         │                                                        │
│         ▼                                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  API Service (instance_service.go)                       │   │
│  │                                                          │   │
│  │  convertInstanceMemoRelatedSettingToStore(               │   │
│  │    v1pb.InstanceSetting_MemoRelatedSetting              │   │
│  │  ) → storepb.InstanceMemoRelatedSetting                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│         │                                                        │
│         ▼                                                        │
│  Store layer receives: storepb.InstanceMemoRelatedSetting       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────┐
│                         Response Flow                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Store layer returns: storepb.InstanceMemoRelatedSetting        │
│         │                                                        │
│         ▼                                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  API Service (instance_service.go)                       │   │
│  │                                                          │   │
│  │  convertInstanceMemoRelatedSettingFromStore(             │   │
│  │    storepb.InstanceMemoRelatedSetting                   │   │
│  │  ) → v1pb.InstanceSetting_MemoRelatedSetting            │   │
│  └──────────────────────────────────────────────────────────┘   │
│         │                                                        │
│         ▼                                                        │
│  Frontend receives: v1pb.InstanceSetting                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Real code example** (from `server/router/api/v1/instance_service.go:246`):

```go
// FROM STORE: storepb → v1pb (used when returning data to client)
func convertInstanceMemoRelatedSettingFromStore(
    setting *storepb.InstanceMemoRelatedSetting,
) *v1pb.InstanceSetting_MemoRelatedSetting {
    if setting == nil {
        return nil
    }
    return &v1pb.InstanceSetting_MemoRelatedSetting{
        DisallowPublicVisibility: setting.DisallowPublicVisibility,
        DisplayWithUpdateTime:    setting.DisplayWithUpdateTime,
        ContentLengthLimit:       setting.ContentLengthLimit,
        EnableDoubleClickEdit:    setting.EnableDoubleClickEdit,
        EnableCustomMemoDate:     setting.EnableCustomMemoDate,
        Reactions:                setting.Reactions,
    }
}

// TO STORE: v1pb → storepb (used when receiving data from client)
func convertInstanceMemoRelatedSettingToStore(
    setting *v1pb.InstanceSetting_MemoRelatedSetting,
) *storepb.InstanceMemoRelatedSetting {
    if setting == nil {
        return nil
    }
    return &storepb.InstanceMemoRelatedSetting{
        DisallowPublicVisibility: setting.DisallowPublicVisibility,
        DisplayWithUpdateTime:    setting.DisplayWithUpdateTime,
        ContentLengthLimit:       setting.ContentLengthLimit,
        EnableDoubleClickEdit:    setting.EnableDoubleClickEdit,
        EnableCustomMemoDate:     setting.EnableCustomMemoDate,
        Reactions:                setting.Reactions,
    }
}
```

**Key pattern:**
- **FromStore**: Converts internal types → public API types (response)
- **ToStore**: Converts public API types → internal types (request)
- Both are **hand-written functions** (not auto-generated)
- Every field must be explicitly mapped

**Why care about these functions?**

If you add a new field to a proto but forget to add it to the conversion function, the data will be silently lost during the conversion. For example:

```go
// ❌ WRONG: Missing field mapping
func convertInstanceMemoRelatedSettingToStore(
    setting *v1pb.InstanceSetting_MemoRelatedSetting,
) *storepb.InstanceMemoRelatedSetting {
    return &storepb.InstanceMemoRelatedSetting{
        DisallowPublicVisibility: setting.DisallowPublicVisibility,
        // ... other fields ...
        // EnableCustomMemoDate is MISSING!
        Reactions:                setting.Reactions,
    }
}
// Result: enableCustomMemoDate is always lost during conversion
```

```go
// ✅ CORRECT: All fields mapped
func convertInstanceMemoRelatedSettingToStore(
    setting *v1pb.InstanceSetting_MemoRelatedSetting,
) *storepb.InstanceMemoRelatedSetting {
    return &storepb.InstanceMemoRelatedSetting{
        DisallowPublicVisibility: setting.DisallowPublicVisibility,
        DisplayWithUpdateTime:    setting.DisplayWithUpdateTime,
        ContentLengthLimit:       setting.ContentLengthLimit,
        EnableDoubleClickEdit:    setting.EnableDoubleClickEdit,
        EnableCustomMemoDate:     setting.EnableCustomMemoDate,  // ← ADDED
        Reactions:                setting.Reactions,
    }
}
```

**Usage in the request flow** (from `instance_service.go:83`):

```go
func (s *APIV1Service) UpdateInstanceSetting(
    ctx context.Context,
    request *v1pb.UpdateInstanceSettingRequest,
) (*v1pb.InstanceSetting, error) {
    // ... auth checks ...

    // 1. Convert v1pb (from client) to storepb (for storage)
    updateSetting := convertInstanceSettingToStore(request.Setting)

    // 2. Store in database
    instanceSetting, err := s.Store.UpsertInstanceSetting(ctx, updateSetting)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to upsert: %v", err)
    }

    // 3. Convert storepb (from storage) back to v1pb (for response)
    return convertInstanceSettingFromStore(instanceSetting), nil
}
```

**Takeaway:**
- The FromStore/ToStore pattern maintains **separation of concerns** between public API and internal storage
- When adding new fields, update **both** the proto definition **and** the conversion functions
- These conversion functions are a common source of bugs when fields are missing

### 5.18 Key Takeaways: Backend Stack

**You now understand:**

1. **Go** - Fast, compiled language with strong typing
2. **gRPC** - RPC framework for calling functions across network
3. **Protocol Buffers** - Binary serialization format + IDL (Interface Definition Language)
4. **.proto file** - Single source of truth that generates types for multiple languages
5. **Connect RPC** - HTTP-compatible gRPC implementation for browsers
6. **Dual protocol** - Expose both Connect (browsers) and REST (external tools) from same backend
7. **Interceptors** - Middleware for gRPC (auth, logging, recovery)
8. **Store layer** - Data access pattern (Repository pattern)
9. **Type safety** - Same types from database to UI (generated from .proto)

**Comparison to what you know:**

| Concept | Node/Express | Go + gRPC + Connect |
|---------|-------------|---------------------|
| API definition | Express routes + Joi/Zod | .proto file |
| Types | TypeScript (frontend), manual (backend) | Auto-generated from .proto |
| Validation | Manual or Joi/Zod | Automatic (protobuf) |
| Serialization | JSON (manual) | Protocol Buffers (auto) |
| Middleware | app.use() | Interceptors |
| Data access | Models/ORM | Store layer |
| Single source of truth | None or OpenAPI | .proto file |

**Real example from Memos** (Go backend):

```go
// server/router/api/v1/memo_service.go
package v1

import (
    "context"
    "github.com/usememos/memos/store"
)

type APIV1Service struct {
    Store *store.Store
    // ... other dependencies
}

// CreateMemo creates a new memo
func (s *APIV1Service) CreateMemo(ctx context.Context, request *v1.CreateMemoRequest) (*v1.Memo, error) {
    // 1. Get current user from context (set by auth interceptor)
    userID := ctx.Value("user_id").(string)

    // 2. Validate request (proto validation is automatic)
    if request.Memo == nil {
        return nil, status.Error(codes.InvalidArgument, "memo is required")
    }

    // 3. Create internal model
    memo := &store.Memo{
        CreatorID: userID,
        Content:   request.Memo.Content,
        Visibility: store.Visibility(request.Memo.Visibility),
    }

    // 4. Call store layer
    createdMemo, err := s.Store.CreateMemo(ctx, memo)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to create memo")
    }

    // 5. Convert to proto type
    return convertToProtoMemo(createdMemo), nil
}
```

### 5.6 Connect RPC vs gRPC-Gateway

Memos uses **dual protocol**:

**Connect RPC** (browser-to-backend):
```typescript
// web/src/connect.ts
import { createConnectRouter } from "@bufbuild/connect";

export const memoServiceClient = createPromiseClient(
  MemoService,
  createConnectTransport({
    baseUrl: "http://localhost:8081",  // From proxy
  })
);

// Usage in component
const { data } = useQuery({
  queryKey: ["memos"],
  queryFn: () => memoServiceClient.listMemos({ pageSize: 50 })
});
```

**gRPC-Gateway** (REST API for external tools):
```bash
# Traditional HTTP/JSON endpoint
curl http://localhost:8081/api/v1/memos?page_size=50

# Returns JSON:
{
  "memos": [...],
  "nextPageToken": "..."
}
```

---

## 6. Frontend-Backend Communication

### 6.1 Request Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         Browser                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐     ┌─────────────────┐                   │
│  │ React Component │ ──▶ │ React Query Hook│                   │
│  └─────────────────┘     └─────────────────┘                   │
│                                  │                              │
│                                  ▼                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │           Connect RPC Client (TypeScript)                │   │
│  │  - Serializes request to binary (protobuf)              │   │
│  │  - memoServiceClient.listMemos({ pageSize: 50 })        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                  │                              │
│                    HTTP POST (binary payload)                   │
│                                  ▼                              │
└──────────────────────────────────────────────────────────────────┘
                            │
                            │ Network
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Go Backend (localhost:8081)                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Connect Interceptor Chain                   │   │
│  │  1. Metadata (adds auth token)                          │   │
│  │  2. Logging (logs request/response)                     │   │
│  │  3. Recovery (handles panics)                           │   │
│  │  4. Auth (validates JWT)                                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                  │                              │
│                                  ▼                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              MemoService (gRPC Service)                  │   │
│  │  - Deserializes binary request                          │   │
│  │  - Calls Store layer                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                  │                              │
│                                  ▼                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Store Layer                            │   │
│  │  - Business logic                                        │   │
│  │  - Database operations                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 6.2 Type Safety Across the Stack

**The beauty of protobuf**: Types are consistent from database to UI!

```protobuf
# Single source of truth (.proto file)
message Memo {
  string name = 1;
  string content = 7;
  Visibility visibility = 9;
}
```

```go
// Generated Go (backend)
type Memo struct {
    Name       string
    Content    string
    Visibility Visibility
}
```

```typescript
// Generated TypeScript (frontend)
export type Memo = Message<"memos.api.v1.Memo"> & {
  name: string;
  content: string;
  visibility: Visibility;
};

// Usage in component - fully type-safe!
const { data } = useQuery({
  queryKey: ["memo", name],
  queryFn: () => memoServiceClient.getMemo({ name }),
});

if (data) {
  console.log(data.content);      // TypeScript knows this is string
  console.log(data.visibility);   // TypeScript knows this is enum
  console.log(data.nonExistent);  // ❌ Compile error!
}
```

---

## 7. State Management Patterns

### 7.1 Two Types of State

**Server State** (from backend):
- Use **React Query** for:
  - Data from API
  - Loading/error states
  - Caching
  - Synchronization

**Client State** (UI-only):
- Use **React Context** for:
  - User preferences
  - UI state (modals, drawers)
  - Temporary form data
  - Authentication status

### 7.2 React Context Pattern

**Real example from Memos** (`web/src/contexts/AuthContext.tsx`):

```typescript
// 1. Define context shape
interface AuthState {
  currentUser: User | undefined;
  userGeneralSetting: UserSetting_GeneralSetting | undefined;
  shortcuts: Shortcut[];
  isInitialized: boolean;
  isLoading: boolean;
}

interface AuthContextValue extends AuthState {
  initialize: () => Promise<void>;
  logout: () => Promise<void>;
  refetchSettings: () => Promise<void>;
}

// 2. Create context (with null for default)
const AuthContext = createContext<AuthContextValue | null>(null);

// 3. Provider component
export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ /* ... */ });

  // Context value with methods
  const value = useMemo(
    () => ({
      ...state,
      initialize: async () => { /* ... */ },
      logout: async () => { /* ... */ },
      refetchSettings: async () => { /* ... */ },
    }),
    [state]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// 4. Custom hook to use context
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}

// 5. Usage in component
function UserProfile() {
  const { currentUser, logout } = useAuth();
  // ^ Fully typed - TypeScript knows currentUser is User | undefined
  //          and logout is () => Promise<void>
}
```

### 7.3 Reducer Pattern for Complex State

**Real example from Memos** (`web/src/components/MemoEditor/state/types.ts` + `reducer.ts`):

```typescript
// types.ts - Define state and actions
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
    isLoading: { saving: boolean; uploading: boolean; loading: boolean };
    isDragging: boolean;
  };
  localFiles: LocalFile[];
}

// Union of all possible actions (discriminated union)
export type EditorAction =
  | { type: "INIT_MEMO"; payload: { content: string; /* ... */ } }
  | { type: "UPDATE_CONTENT"; payload: string }
  | { type: "SET_METADATA"; payload: Partial<EditorState["metadata"]> }
  | { type: "ADD_ATTACHMENT"; payload: Attachment }
  | { type: "REMOVE_ATTACHMENT"; payload: string }
  | { type: "TOGGLE_FOCUS_MODE" }
  | { type: "SET_LOADING"; payload: { key: LoadingKey; value: boolean } }
  | { type: "RESET" };
```

```typescript
// reducer.ts - Implement state transitions
export function editorReducer(state: EditorState, action: EditorAction): EditorState {
  switch (action.type) {
    case "UPDATE_CONTENT":
      return { ...state, content: action.payload };

    case "SET_METADATA":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          ...action.payload,  // Partial merge
        },
      };

    case "ADD_ATTACHMENT":
      return {
        ...state,
        metadata: {
          ...state.metadata,
          attachments: [...state.metadata.attachments, action.payload],
        },
      };

    case "TOGGLE_FOCUS_MODE":
      return {
        ...state,
        ui: { ...state.ui, isFocusMode: !state.ui.isFocusMode },
      };

    case "RESET":
      return initialState;

    default:
      return state;
  }
}
```

```typescript
// Usage in component
import { useReducer } from "react";

function MemoEditor() {
  const [state, dispatch] = useReducer(editorReducer, initialState);

  // Dispatch actions
  const handleContentChange = (content: string) => {
    dispatch({ type: "UPDATE_CONTENT", payload: content });
  };

  const handleAddAttachment = (attachment: Attachment) => {
    dispatch({ type: "ADD_ATTACHMENT", payload: attachment });
  };

  return (
    <>
      <textarea
        value={state.content}
        onChange={(e) => dispatch({ type: "UPDATE_CONTENT", payload: e.target.value })}
      />
      <button onClick={() => dispatch({ type: "TOGGLE_FOCUS_MODE" })}>
        Focus Mode
      </button>
    </>
  );
}
```

### 7.4 Combining React Query + Context + Reducer

```typescript
function MemoEditor({ memoName }: { memoName?: string }) {
  // React Query for server data
  const { data: memo } = useMemo(memoName ?? "", {
    enabled: !!memoName,
  });

  // Reducer for editor state
  const [state, dispatch] = useReducer(editorReducer, initialState);

  // Mutation for saving
  const { mutate: updateMemo } = useUpdateMemo();

  // Initialize editor when memo loads
  useEffect(() => {
    if (memo) {
      dispatch({
        type: "INIT_MEMO",
        payload: {
          content: memo.content,
          metadata: {
            visibility: memo.visibility,
            attachments: memo.attachments,
            relations: memo.relations,
          },
          timestamps: {
            createTime: memo.createTime?.toDate(),
            updateTime: memo.updateTime?.toDate(),
          },
        },
      });
    }
  }, [memo]);

  // Save handler
  const handleSave = () => {
    updateMemo({
      update: {
        name: memoName,
        content: state.content,
        visibility: state.metadata.visibility,
      },
      updateMask: ["content", "visibility"],
    });
  };

  return <textarea /* ... */ />;
}
```

---

## 8. Quick Reference: TypeScript Types

| Type | Description | Example |
|------|-------------|---------|
| `Partial<T>` | Make all properties optional | `Partial<User>` |
| `Required<T>` | Make all properties required | `Required<User>` |
| `Pick<T, K>` | Select specific properties | `Pick<User, "id" \| "name">` |
| `Omit<T, K>` | Remove specific properties | `Omit<User, "password">` |
| `Record<K, V>` | Object with key/value types | `Record<string, number>` |
| `Readonly<T>` | Make properties readonly | `Readonly<User>` |
| `readonly` | Readonly array/tuple | `readonly string[]` |
| `as const` | Create literal type | `["a", "b"] as const` |
| `keyof T` | Get keys as union type | `keyof User` = `"id" \| "name"` |
| `typeof T` | Get type from value | `typeof config` |
| `T \| K` | Union type (OR) | `string \| number` |
| `T & K` | Intersection (AND) | `Name & Age` |
| `T[]` | Array | `number[]` |
| `Array<T>` | Generic array | `Array<number>` |
| `[T, U]` | Tuple | `[string, number]` |
| `T?` | Optional | `name?: string` |
| `T extends U` | Generic constraint | `<T extends string>` |
| `infer T` | Infer type | `infer ReturnType` |

---

## 9. Common Patterns Cheat Sheet

### 9.1 Axios → React Query Migration

| Axios | React Query |
|-------|-------------|
| `axios.get("/api/users")` | `useQuery({ queryKey: ["users"], queryFn: () => fetch("/api/users") })` |
| `useState(data)` | `data` from `useQuery` |
| `useState(loading)` | `isLoading` from `useQuery` |
| `useState(error)` | `error` from `useQuery` |
| `useEffect(() => fetch(), [])` | `useQuery` (auto-fetches) |
| Manual caching | Automatic with `queryKey` |
| Manual refetch | `refetch()` from `useQuery` |

### 9.2 Type Guards

```typescript
// Narrowing with typeof
function process(value: string | number) {
  if (typeof value === "string") {
    value.toUpperCase(); // ✅ string
  } else {
    value.toFixed(2);    // ✅ number
  }
}

// Narrowing with instanceof
class Dog { bark() {} }
class Cat { meow() {} }

function sound(animal: Dog | Cat) {
  if (animal instanceof Dog) {
    animal.bark();  // ✅ Dog
  } else {
    animal.meow();  // ✅ Cat
  }
}

// Narrowing with in
interface Bird { fly: () => void; }
interface Fish { swim: () => void; }

function move(animal: Bird | Fish) {
  if ("fly" in animal) {
    animal.fly();   // ✅ Bird
  } else {
    animal.swim();  // ✅ Fish
  }
}

// Discriminated unions (type property)
interface Success { type: "success"; data: string; }
interface Error { type: "error"; message: string; }

function handle(result: Success | Error) {
  if (result.type === "success") {
    console.log(result.data);    // ✅ Success
  } else {
    console.log(result.message); // ✅ Error
  }
}
```

### 9.3 React Query + TypeScript Patterns

```typescript
// Type-safe query factory
export const userKeys = {
  all: ["users"] as const,
  detail: (id: string) => [...userKeys.all, "detail", id] as const,
} as const;

// Type-safe query hook
export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: async () => {
      const response = await userServiceClient.getUser({ name: id });
      return response;
    },
  });
}

// Type-safe mutation with callbacks
export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (user: User) => {
      return await userServiceClient.updateUser({ user });
    },
    onSuccess: (user) => {
      // TypeScript knows user is User
      queryClient.setQueryData(userKeys.detail(user.name), user);
    },
    onError: (error) => {
      // TypeScript knows error is unknown (cast if needed)
      console.error("Failed to update user:", error);
    },
  });
}
```

---

## 10. Key Takeaways

### TypeScript Concepts
1. **`Partial<T>`** - Make properties optional (common in updates)
2. **`Pick<T, K>` / `Omit<T, K>`** - Select/remove properties
3. **Union types (`\|`)** - Value can be one of several types
4. **Intersection types (`&`)** - Combine multiple types
5. **Generics (`<T>`)** - Parameterized types for reuse
6. **`as const`** - Create readonly literal types

### React Query Patterns
1. **Query keys** - Unique identifiers for cache
2. **Query factories** - Type-safe key builders
3. **Mutations** - For server modifications
4. **Optimistic updates** - Instant UI, rollback on error
5. **Invalidation** - Refresh stale data

### Go + gRPC vs Node/Express
1. **Protocol Buffers** - Binary, faster than JSON
2. **Code generation** - Auto-generated types from `.proto`
3. **Type safety** - End-to-end from DB to UI
4. **Dual protocol** - Connect RPC (browsers) + REST (external)

### Architecture Patterns
1. **Server state** - React Query
2. **Client state** - React Context + useState
3. **Complex state** - useReducer
4. **Type safety** - Generated from protobuf

---

## 11. Learning Path

### Step 1: Master TypeScript Basics
- Understand types, interfaces, generics
- Learn utility types (`Partial`, `Pick`, `Omit`, `Record`)
- Practice type guards and narrowing

### Step 2: React Query Fundamentals
- Replace `useEffect` + `axios` with `useQuery`
- Learn query keys and caching strategies
- Master mutations and optimistic updates

### Step 3: Read This Codebase
- Start with hooks: `web/src/hooks/`
- Understand the types: `web/src/types/proto/`
- Study the patterns: `web/src/contexts/`

### Step 4: Explore Backend (Optional)
- Read `.proto` files: `proto/api/v1/`
- Check services: `server/router/api/v1/*_service.go`
- Compare with Express/FastAPI patterns you know

---

## File Reference

| Concept | File Reference | Description |
|---------|---------------|-------------|
| React Query hooks | `web/src/hooks/useMemoQueries.ts` | Complete query/mutation patterns |
| Query keys | `web/src/hooks/useMemoQueries.ts:10-17` | Type-safe key factory |
| Optimistic updates | `web/src/hooks/useMemoQueries.ts:80-124` | Update + rollback pattern |
| Context pattern | `web/src/contexts/AuthContext.tsx` | Auth state management |
| Reducer pattern | `web/src/components/MemoEditor/state/` | Complex state with reducer |
| TypeScript types | `web/src/types/proto/api/v1/` | Generated protobuf types |
| Protocol buffers | `proto/api/v1/memo_service.proto` | API contract definition |
| Go service | `server/router/api/v1/memo_service.go` | Backend implementation |
