Palette & Styling Guide
=======================

Purpose
-------
This package centralizes terminal color and style decisions for the project. Use the provided palette roles and Style wrappers to keep UI output consistent and easy to change.

Key Concepts
------------
- ColorPalette (type ColorPalette): holds ANSI sequences for named roles (Success, Error, DiffBadgeAdd, etc.).
- DefaultColors: the package-level ColorPalette initialized by DefaultPalette().
- style.Style values (e.g., style.Success, style.TableHeader): convenient wrappers created from DefaultColors and used across the codebase. They expose Render(text string) string.

How to use
----------
- For one-off text rendering use the Style wrapper:

    style.Success.Render("OK")

- For raw ANSI sequences (rare) read from the palette:

    fmt.Print(style.DefaultColors.Success + "OK" + style.Reset)

- Prefer using Style.Render where possible — it keeps resets and semantics consistent.

Combined roles
--------------
If you find yourself concatenating two palette fields repeatedly (for example Header + TableStatusAdd), add a combined role to ColorPalette (e.g., HeaderKindAdd) and initialize it in DefaultPalette. This keeps the API surface small and the patterns explicit.

Adding a new role
-----------------
1. Add a field to ColorPalette in pkg/style/palette.go.
2. Initialize it in DefaultPalette().
3. Create a style wrapper in pkg/style/styles.go if you need style.Render convenience:

    // in styles.go
    MyNewRole = NewStyle(DefaultColors.MyNewRole)

4. Update call sites to use the new role via style.MyNewRole.Render(text).

Testing
-------
Run the full test suite after any change:

    go test ./...

Notes
-----
- Avoid calling Render on fields of DefaultColors (they are strings). Use the Style variables defined in styles.go (they wrap DefaultColors and expose Render).
- Keep changes small and localized; run tests after each migration.
