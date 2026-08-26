# vhs docs/demo.tape  — run via `make demo` (builds gitpad and the demo repo first).
# Produces docs/demo.gif plus still screenshots in docs/screenshots/.

Output docs/demo.gif

Set Shell zsh
Set FontFamily "Menlo"
Set FontSize 15
Set Width 1640
Set Height 920
Set Padding 14
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set Framerate 24
Env COLORTERM truecolor

Hide
Type 'cd "$GITPAD_DEMO" && clear && gitpad' Enter
Sleep 2.5s
Show

# The log: branches, graph, changes, details.
Sleep 2s
Screenshot docs/screenshots/log.png
Down@150ms 8
Sleep 1s

# Commit context menu (Enter or right-click).
Enter
Sleep 1.5s
Screenshot docs/screenshots/menu.png
Down@250ms 3
Sleep 800ms
Escape
Sleep 600ms

# Diff viewer from the Changes pane.
Type "3"
Sleep 400ms
Enter
Sleep 1.5s
Screenshot docs/screenshots/diff.png
Down@120ms 6
Sleep 800ms
Escape
Sleep 500ms

# Commit workspace: checklist, message, live diff.
Type "2"
Type "g"
Sleep 400ms
Type "C"
Sleep 1.5s
Down@300ms 2
Space
Sleep 600ms
Enter
Sleep 400ms
Type "feat(search): debounce suggestions on the client"
Sleep 1.5s
Screenshot docs/screenshots/commit.png
Sleep 800ms
Escape
Sleep 300ms
Escape
Sleep 600ms

# Push dialog.
Type "P"
Sleep 2s
Screenshot docs/screenshots/push.png
Sleep 1.5s
Escape
Sleep 1s
Type "q"
Sleep 500ms
