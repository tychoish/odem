---
name: add-minutes-query
description: |
  Add a new query of the minutes data to the `odem` system. Given a description or name of the operation, implement, test, and integrate it. Expose the query in the fuzzy CLI (pkg/fzfui) for interactive use.
---

## Step 1 — Understand the schema

Read `pkg/db/fasoladb/minutes_schema.sql`, `pkg/db/views.sql`, and existing queries in `pkg/db/db.go` before writing SQL. Prefer views over raw tables.

Key tables/views:

| Name                                                  | Alias | Key columns                                                             |
|-------------------------------------------------------|-------|-------------------------------------------------------------------------|
| `song_leader_joins`                                   | slj   | `song_id`, `leader_id`, `minutes_id`, `lesson_id`                       |
| `leader_song_stats`                                   | lss   | `leader_id`, `song_id`, `lesson_count`, `lesson_rank`                   |
| `song_stats`                                          | ss    | `song_id`, `year`, `lesson_count`, `rank`                               |
| `book_song_joins`                                     | bsj   | `song_id`, `book_id`, `page_num`, `keys` — use `book_id = 2` for Denson |
| `leaders`                                             |       | `id`, `name`, `lesson_count`                                            |
| `leader_name_invalid`                                 | inv   | exclude with `LEFT JOIN … WHERE inv.name IS NULL`                       |
| `minutes`                                             |       | `id`, `Name`, `Year`, `Date`, `Location`                                |
| `locations`                                           |       | `id`, `state_province`, `country`, `city`                               |
| `minutes_location_joins`                              | mlj   | `minutes_id → location_id`                                              |
| `leader_coattendance`                                 | lca   | `leader_a_id`, `leader_b_id`, `shared_singings` (precomputed)           |
| `song_details`, `lesson_details`, `song_leader_stats` |       | precomputed views — check columns before use                            |

## Step 2 — Validate the SQL

Run the query against `/tmp/minutes.db` using the `sqlite3` CLI before writing any Go. Use `"Bud Oliver"`, `"Sam Kleinman"`, `"Rose Altha Taylor"` for leader-scoped queries; `"82t"`, `"475"`, and `"89"` for song-scoped queries.

**Ask the user to confirm the results look correct before continuing.**

## Step 3 — Add the method to `pkg/db/db.go`

**Fixed query:**
```go
func (conn *Connection) MyQuery(ctx context.Context, name string, limit int) iter.Seq2[models.LeaderSongRank, error] {
    const query = `SELECT …`
    cur, err := conn.db.QueryContext(ctx, query, name, cmp.Or(limit, 40))
    if err != nil {
        return irt.Two(models.LeaderSongRank{}, err)
    }
    return dbx.Cursor[models.LeaderSongRank](cur)
}
```

**Dynamic query — use `dbx.Builder`, never `fmt.Sprintf` + `strings.Repeat`:**

```go
func (conn *Connection) MyQuery(ctx context.Context, limit int, items ...models.SingingLocality) iter.Seq2[models.LeaderSongRank, error] {
    strs := make([]string, len(items)) // named string types must be converted to []string
    for i, s := range items { strs[i] = string(s) }

    var qb dbx.Builder
    qb.WithSQL(`SELECT …`)
    if len(strs) > 0 {
        qb.With(" WHERE col IN (%+?)", strs) // %+? expands slice to ?, ?, ?
    }
    qb.WithSQL(` ORDER BY count DESC`)
    if limit > 0 {
        qb.With(" LIMIT %?", limit)
    }
    query, args := qb.Build()
    cur, err := conn.db.QueryContext(ctx, query, args...)
    if err != nil {
        return irt.Two(models.LeaderSongRank{}, err)
    }
    return dbx.Cursor[models.LeaderSongRank](cur)
}
```

`dbx.Builder` placeholders (SQLite): `%?` scalar, `%+?` slice (`[]string`, `[]int`, `[]any` — not named types), `%s` raw SQL fragment.

`dbx.Cursor` maps columns to struct fields by `db:` tag; unmatched fields are zero. `models.LeaderSongRank` tags: `rank` (int), `name`, `count`, `song_page`, `song_title`, `song_keys` (strings), `ratio` (float64).

## Step 4 — Write a smoke test in `pkg/db/db_test.go`

```go
func TestMyQuery(t *testing.T) {
    conn, ctx := testConn(t)
    count := 0
    for _, err := range conn.MyQuery(ctx, testLeader, 5) {
        if err != nil { t.Fatal(err) }
        count++
    }
    if count == 0 {
        t.Errorf("MyQuery(%q): expected at least one result", testLeader)
    }
}
```

Add `break` after the first iteration for unbounded queries. Assert domain invariants where meaningful (e.g. ratios in (0,1]). Run `go test ./pkg/db/` and **ask the user to confirm the output looks correct before continuing.**

## Step 5 — Add an action to `pkg/fzfui/actions.go`

```go
func myQueryAction(ctx context.Context, dbconn *db.Connection, singer string) error {
    singer, err := interactivelyResolveSingerName(ctx, dbconn, singer)
    if err != nil { return err }
    grip.Infof("…", singer)
    return renderTopLedSongs(dbconn.MyQuery(ctx, singer, 32))
}
```

- `interactivelyResolveSingerName` — prompts for a leader name if not provided
- `renderTopLedSongs(seq)` — tabular display for `iter.Seq2[models.LeaderSongRank, error]`
- `renderTopLeaders(ctx, conn, pageNum)` — for top-leaders-of-a-song
- For `KV` results or custom shapes, build a `tabby.New()` table directly (see existing examples in the file)
- For multi-select inputs (e.g. localities): `erc.FromIteratorAll(infra.NewFuzzySearch[T](options).Find("prompt"))`

## Step 6 — Register in `pkg/clidispatch/dispatcher.go`

Four edits, all in `dispatch.go`:

1. **`iota` block** — add before `MinutesAppOpInvalid`:
   ```go
   MinutesAppOpMyOp
   MinutesAppOpInvalid
   ```

2. **`GetInfo()`** — key is the kebab-case CLI name, value is the usage string:
   ```go
   case MinutesAppOpMyOp:
       return irt.MakeKV("my-op", "one-line description")
   ```

3. **`Dispatch()`**:
   ```go
   case MinutesAppOpMyOp:
       return myQueryAction(ctx, conn, strings.Join(args, " "))
   ```

4. **`NewMinutesAppOperation()`** — canonical name plus aliases:
   ```go
   case "my-op", "my-op-alias":
       return MinutesAppOpMyOp
   ```

`String()`, `Commander()`, and `AllMinutesAppCommanders()` all derive from `GetInfo()` — no other changes needed.

## Step 7 — Add a Static Report rendering  to `pkg/reportui/reports.go`

```go
mb.H2("My New Section")
mb.Paragraph("One-line description.")
// render — choose one:
writeSongTable(&mb, erc.HandleAll(conn.MyQuery(ctx, singer, 25), ec.Push))                          // iter.Seq2[models.LeaderSongRank, error]
mb.KVTable(irt.MakeKV("Name", "Count"),                                                             // iter.Seq2[irt.KV[string, int], error]
    irt.Convert2(irt.KVsplit(erc.HandleAll(conn.MyQuery(ctx, singer, 25), ec.Push)), intValToStr))
writeMyTable(&mb, erc.HandleAll(conn.MyQuery(ctx, singer, 25), ec.Push))                            // custom type — add helper at bottom of file
```

For custom types add a table helper alongside `writeSongTable`. `mdwn.Builder` tables take `iter.Seq` — use `erc.HandleAll` to strip errors.

## Step 8 - Register all functions in `pkg/clidispatch/dispatcher.go` 

New operations must be added in the following switch statements:

- `clidispatch.MinutesAppOperation.ReportsDispatcher()` (wire up `reportui` operations)
- `clidispatch.MinutesAppOperation.FuzzyMatcher()` (wire `fzfui` operations)
- `clidispatch.MinutesAppOperation.GetInfo()` to describe the name of the entry point and usage text.
- `clidispatch.NewMinutesAppOperation` add logical command aliases.

## Step 9 — Build and verify

Run `go build ./...` and fix any errors. Use `go test ./... -timeout=1m` to verify that all tests pass. Confirm the operation appears in the fzf menu and returns sensible output.
