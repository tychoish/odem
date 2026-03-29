# `odem` -- a command line (& etc.) tool for Sacred Harp Singing Minutes Data

## Background

Taking the work Mark has done with the minutes data for Sacred Harp singing, I wanted to put together a library/app(lication) for working with the minutes data at a (potentially) higher level (than writing and running queries directly against the database.)

This is entirely for fun, and made possible by Mark's [fasolaminutes_parsing](https://github.com/marktgodfrey/fasolaminutes_parsing/) (and of course everyone who takes minutes and sings Sacred Harp).

Allowing to the fact that this isn't running on a phone _and_ doesn't need to be downloaded over a mobile network, it runs some queries that would either take too long _or_ require storing the data in a way that wouldn't otherwise be viable. The queries are bordering on the absurd and to make (most) operations fast, I've done some wacky database that take up a bunch of (temporary) on the system it runs on. It does many of the things that the Phone Minutes App does, with some extra things: Some of the highlights:

- _singing buddies_: who have you been at the most singings with.

- (surprising?) _singing strangers_: who have you _never_ sung with (but who has sung with most (man?) of the people that you've sung with!)

- singers' "_connectedness_" or the percentage of total singers that you've been at singings with.

- _unfamilar hits_: the most popular songs that are underrepresented at the singings you've been at.

- _never sung_: songs that have never been sung at a singing you've been at.

- _locally popular_: the most popular songs in your geographic region (state, etc.).

- _popular in your experience_: songs that have been sung at singings you've been to, ordered by experience.

- _leading role models_: the list if your most frequent leads with the most frequent leaders of those songs added in.

- _leader share_: the percentage of all of the leads in the database that a specific leader has led (and a list of all leaders ordered by number of leads, with the corresponding _cumulative percentage_.)

... and more.

There's a command line interface with a fuzzy-search interface, as well as the ability to generate markdown files with "reports," for every query. Coming soon:

- packing/delivering the application in various ways so that more people can use it, (if they wanted.)

- some improvements to the fuzzy search experience to make it feel more like an application and less like a weird command line thing.

- more queries and views!

## (possibly) FAQ

- _Why `odem`?_ I wanted to name the program after a sacred harp song, but I have projects named after `jasper` (public,) and `sardis` (private). I mostly chose it because it has short and it's easy to type (letters are split between each hand on a qwerty keyboard.)

- _Why a CLI?_ Most of my computing work happens in CLI and adjacent contexts, so this makes sense for me and how I use computers. Also broadly speaking, I'm much more "build a CLI" and less of "make an app" or "make a website" kind of software engineer.

## The Fine Print

The reset of this file contains a bunch of technical details about the program, running/developing the software. Feel free to stop here.

Having said that, I don't think you have to be a programmer to use
`odem, and I've tried to be detailed in the documentation/instructions
below for installing and getting started, so feel free to stick around. 😃

The application is a fairly modular and largely standard Go application. The database (sqlite) is pulled from Mark's repository (as a git submodule) and attached to the binary, and then written out to your temporary directory (e.g. `/tmp`) when the program runs for the first time/as needed. The database bootstrapping takes a bit under 90 seconds (on my not very impressive laptop,) so be ready for that.

Bootstrapping notwithstanding, the application is (relatively speaking,) very self contained: there's one file, it does its own setup and it will rebuild the database after upgrades. The fuzzy search interfaces make it much easier than your average CLI.

I plan to have pre-built artifacts at some point in the future if pepole are interested in running it themselves, in the mean time, if you're not used to this kind of thing, this might be a lot. Apologies...

### Setup

1. You need to install Go. Do that according to [these direction](https://go.dev/doc/install).

2. You need to have a working terminal with git. I haven't tested this on Macs but it should be fine. There's nothing keeping this from working on Windows, particularly WSL, but I know less of the details.

   Open the terminal and run the `git` command. If it prints a bunch of stuff that starts with `usage` then you're good.

3. Clone the repository and the submodule with the following command:

   ```bash
   git clone https://github.com/tychoish/odem.git --recursive-submodules
   ```

   The `--recursive-submodules` option downloads the database from Mark's repository and gets it all set up. If you omit this option above or clone the repository in another way, and you don't have the datebase (run `ls pkg/db/fasoladb` to check), you can setup the submodule with:

   ```bash
   git submodule sync
   git submodule update
   ```

3. Build the binary. Use the following command to build the application:

   ```bash
   go build ./cmd/odem.go
   ```

   At this point, you can run the program:

   ```bash
   ./odem --help
   ./odem fzf --help
   ./odem report --help
   ```

   The first time you actually access the data, the application will write out the database. This will also happen any time your temporary space is cleared (typically on reboot), you upgrade (e.g. run `go build` as above.)

4. **Optional**: Make `odem` accessible from any terminal. I prefer to do this by creating a symbolic link in `/usr/local/bin/` to the `odem` artifact in this directory. There are other ways, and this is optional but:

   ```bash
   sudo ln -s $(pwd)/odem /usr/local/bin/odem
   ```

	Now from any system prompt on your system (assuming `/usr/local/bin` is in your (search) `$PATH`), you can run the `odem` command and everything should work.

   This only needs to be done once. If it doesn't work your link may not be in the search path (or the build may not exist). Run `echo $PATH` to see the list (`:` separated) of paths. You can add `/usr/local/bin` to your search(for the local session _only_) with:

   ```bash
   export PATH="$PATH:/usr/local/bin/"
   ```

   The procedure to make this permanent is probably beyond the scope of this document, and depending on your system and configuration.

   At some point soon, hopefully, I'll have an installation procudre with fewer sharp edges.

4. You can write the database to the temporary space explicitly:

   ```bash
   odem setup
   ```

   If you want to delete the database, the following operation works:

   ```bash
   odem setup reset
   ```

### Development

#### A Guide to the Codebase

Let's start with a guide to the codebase:

- `cmd/ep/` the entry points for various subcommands. The `odem.go` file only registers the top-most level of subcommands: for the most part the `cmdr` package makes it possible to have the barest minimum of CLI handling code. There are some helper functions (with macro vibes) in `pkg/infra/` for the moment.

- `pkg/db` has all of the more complex queries. All functions take parameters, and return `iter.Seq2[T, error]` type iterators. Queries are function-scoped constants. The `pkg/db/fs.go` file contains the database setup, initialization, and management of local database.

- `pkg/models` includes data types generated from the database using `sqlc` (and `sqlc` generated queries.) The `usr.go` file contains hand-generated models in support of complex queries.

- `odem` uses `sqlc` for only two purposes:

  1. Some simple "Find One"-type queries, but mostly...

  2. To generate Go models from the database schema, and related views. The `pkg/db/views.sql` file, contains a number of additional views (which sqlc` will generate a type for.)

  Most of the queries are handled in the `pkg/db` package.

- for the CLI handling the `pkg/dispatch/` contains the registry and dispatcher for command line entry points, with the rendering of the reports in the `pkg/reportui` and the fzf-inspired interface in `pkg/fzfui`. `dispatch` has the switch statements

Basically, `reportui` and `fzfui` have a bunch of functions that handle user interaction and rendering of data that comes from the `db` package. The `dispatch` package sits between user requests and those handlers, and that's it.

#### AI Coding Assistants

It's 2026, and one of my (personal) goals with this project has been to figure out how and to what extent I could get to use AI tools to in the development of this project. So there's some scaffolding here.

Agents have written many of the queries, and are responsible for much of the database optimization, and test code. The architecture and code organization was manual.

The process of adding a new query is characterized as an agent skill as `add-minutes-query`. If you direct the agent to use this skill with a tight/clear description of the analysis, it does a reasonable job of producing a new operation.

In genreal my take, for the moment, is that agents are great for:

- writing tests, both during development and when fixing bugs to prevent future regressions.

- characterizing and fixing bugs, particularly the ones that are rooted in bad queries, and sometimes ones introduced by manual refactors.

- writing deterministic tool, typically somewhat collaboratively with an organic programmer.

In persuit of this, I tend to scrutinize anything written by Agents, and also typically work in a fairly bottom up style, and I tend to decompose problems to some extent before delegating them to an agent. While I have long been a skeptic about coding agents, and AI in general, these tools do a pretty good job of generating SQL queries, and for developing a robust test suite: at the same time their ability to "design"/generate a reasonable architecture or interfaces is somewhat suspect.

As always short prompts that point out prototypes or patterns that you want to use; pointers to where code for various components or connections, as well as how you want tests to be written has proven more effective.

#### The `tychoish` Ecosystem

This project uses a collection of other tools and libraries that I've written that aren't common in Go programs outside of stuff I work on, in particular:

- [github.com/tychoish/fun](https://github.com/tychoish/fun), which is a collection of generic container types, function-object-wrappers, error collection, and some ergonomic programming tools. It's opinionated and a bit quirky.

  In particular the `irt` library provides tooling for working with go's (newish) native iterator types. This leads to a programming style that's a bit more "lispy" and lazy/iterator driven, and function-centered than the mainline go conventions.

  Also the `strut` package (for **str**ing **ut**ilities) package provides some higher-level interaces around `strings.Builder`,  `bytes.Buffer`, (and `Mutable` for providing a pooled, string-ish type around around a `[]byte` slice.) This all makes building text nd string output more ergonomic and higher-level.

- [github.com/tychoish/grip](https://github.com/tychoish/grip), a structured logging library with a focus on ease-of-use, flexibility, and ergonomics.

- [github.com/tychoish/dbx](https://github.com/tychoish/dbx), is an elaboration on the `go-simpler.org/queries` library, with expanded support for decoding database-tuples into go types and a more optimized query builder.

- [github.com/tychoish/cmdr](https://github.com/tychoish/cmdr) provides a thin layer on top of the `urfave/cli` package for handling dispatching of CLI applications. The goal is the minimize the amount of boilerplate/overhead for arg-parsing, and also to maintain a more robust boundary between the `cli` package and any application code (_without sacrificing the support of the type system._)

#### Roadmap

This is all very rough, and mostly to provide a vague idea of the kinds of things I'm interested in doing.

Additional UIs:

- MCP serer (why not?)

- Telegram Bot (for remote access?) and fun.

- Minutes data guessing games (for road trips.)

- mostly for giggles, I'd like to see about building an emacs wrapper around this using [consult](https://github.com/minad/consult) and [marginalia](https://github.com/minad/marginalia) maybe with a little tabular mode? 🤷 🖥️

Other features:

- Super-locality queries (e.g. multi-state regions.)

- "Inactive singers" (no activity in N-years,) filters so we can have "most common leader who's still singing."

- Statefull "memory" of who a given user is, to provide relative statistics between two or small groups of singers.

- ratio of leads of a particular song to the leaders entire set of leads.

Improvements:

- the fuzzy interface is (today) mostly a menu system to generate markdown fragments, could operate more as a loop so you could jump through data in the fuzzy interface.

- the `reportui` package collapses selection/resolution and rendering and this should be more divided. Potentially (if this makes sense) rendered in a way that lets those forms be reused.

- most of the limits and other parameters for queries are not (particularly) exposed to queries, could improve this.

- while the fuzzy search tool is pretty good, it's slightly less flexible in some ways than I'd like, and I might explore replacing the library that provides this facility.

### Contribution

Definitely down to have other folks contribute with this... I'm particularly interested in this being a jumping off point for singers who are interested in programming/software/data things but who don't think of themselves as programmers. Let's talk about it!

Looking forward to singing with you all soon!
