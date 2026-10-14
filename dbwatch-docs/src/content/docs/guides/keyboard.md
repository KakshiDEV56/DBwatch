---
title: Keyboard and mouse
description: Vim-style movement, search, read mode, and the mouse. The same actions work from the in-app guide on ?.
---

Focus moves between three places: the panel sidebar, the database table, and the logs. Keys apply to whichever place is focused, except the ones listed as global.

## Move

| Key | Action |
| --- | --- |
| `h` / `l` | Move focus between panels, the database table, and logs. Tab and Shift+Tab do the same. |
| `j` / `k` | Move within the focused section. Arrow keys work too. |
| Enter | On a database row, select it. On a log row, open the detail view. |
| `Esc` | Close the detail view, the guide, or search. On the add-database prompt, step back. On the welcome screen, quit. |
| `G` | In the logs or the guide, jump to the end. `g` in the guide jumps to the top. |

## Act

| Key | Action |
| --- | --- |
| `a` | Add a database |
| `d` | Remove the selected database, after confirmation |
| `c` | Copy the selected log row, or the open detail view |
| `/` | Open search |
| `n` / `N` | Next or previous search match |
| `m` | Toggle read mode and live mode |
| `r` | Poll immediately |
| `?` | Open or close the in-app guide |
| `q` | Quit. Ctrl+C also quits. |

## Search

`/` opens a centered prompt. What you type filters the current panel's log and highlights matching database names. Enter confirms the query. `Esc` closes the prompt and leaves the previous query as it was. `n` and `N` move the log cursor to the next or previous match and stop auto-follow, so a new event does not yank the view.

## Read mode

Live mode polls every database on the interval. Read mode (`m`) stops that. The screen holds the last sample so you can read a query or a lock without it being replaced. Press `m` again to resume. The next tick polls immediately.

## Mouse

The scroll wheel moves the selection in the section under the pointer. Clicking a sidebar card, a database row, or a log row selects it, the same as `j` and `k`.

## Copy

`c` copies the selected log entry or the open detail view, including the full query text when there is one. The copy is OSC 52, written to the terminal. It works over SSH. It does not need `xclip`, `pbcopy`, or a window system.
