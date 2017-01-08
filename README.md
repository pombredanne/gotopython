A work-in-progress Go to Python transpiler. 
It can currently only compile toy programs.

# Example usage

This will compile the package at path `./mypackage` into a Python module `mypackage.py`:

```
gotopython -o mypackage.py ./mypackage
```

# Implementation status

The parts of the Go language spec that are implemented are:

| Expression     | Example                   | Implemented |
|----------------|---------------------------|-------------|
| BadExpr        |                           | n/a         |
| Ident          | `myVar`                   | ✓           |
| Ellipsis       | `...`                     |             |
| BasicLit       | `42`                      | ✓           |
| FuncLit        | `func(t T) {}`            |             |
| CompositeLit   | `&T {x: 1, y: 2}`         | ✓           |
| ParenExpr      | `(x)`                     |             |
| SelectorExpr   | `x.y`                     | ✓           |
| IndexExpr      | `x[y]`                    | ✓           |
| SliceExpr      | `x[y:z]`                  | ✓           |
| TypeAssertExpr | `x.(T)`                   |             |
| CallExpr       | `x(y,z)`                  | ✓           |
| StarExpr       | `*x`                      |             |
| UnaryExpr      | `-x`                      | ✓           |
| BinaryExpr     | `x+y`                     | ✓           |
| KeyValueExpr   | `x: y`                    | ✓           |
| ArrayType      | `[]T`                     | partial     |
| StructType     | `struct { T x }`          |             |
| FuncType       | `func(T) U`               |             |
| InterfaceType  | `interface {}`            |             |
| MapType        | `map[T]U`                 |             |
| ChanType       | `chan<- T`                |             |

| Statement      | Example                     | Implemented |
|----------------|-----------------------------|-------------|
| BadStmt        |                             | n/a         |
| DeclStmt       | `var x T`                   |             |
| EmptyStmt      |                             |             |
| LabeledStmt    | `label: ...`                |             |
| ExprStmt       | `x`                         | ✓           |
| SendStmt       |                             |             |
| IncDecStmt     | `x++`                       | ✓           |
| AssignStmt     | `x, y := z`                 | ✓           |
| GoStmt         | `go f()`                    |             |
| DeferStmt      | `defer f()`                 |             |
| ReturnStmt     | `return x, y`               | ✓           |
| BranchStmt     | `break`                     | ✓           |
| BlockStmt      | `{...}`                     | ✓           |
| IfStmt         | `if x; y {...}`             | ✓           |
| CaseClause     | `case x>y:`                 | ✓           |
| SwitchStmt     | `switch x; y {...}`         | partial     |
| TypeSwitchStmt | `switch x.(type) {...}`     |             | 
| CommClause     |                             |             |
| SelectStmt     |                             |             |
| ForStmt        | `for x; y; z {...}`         | ✓           |
| RangeStmt      | `for x, y := range z {...}` | ✓           |

| Spec       | Example      | Implemented |
|------------|--------------|-------------|
| ImportSpec | `import "x"` |             |
| ValueSpec  | `var x T`    |             |
| TypeSpec   | `type T U`   | ✓           |

| Language feature | Implemented |
|------------------|-------------|
| `panic`          |             |
| package unsafe   |             |
| goroutines       |             |
| Imports          |             |
| Name collisions  |             |
| Scoping rules    |             |
| `fallthrough`    |             |
| `goto`           |             |
| cgo              |             |