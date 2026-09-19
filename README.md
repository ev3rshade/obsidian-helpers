# Go Obsidian Helpers

This project is an exercise to help me learn Go, version control, test construction, and conceptualize methods to maintain my Obsidian Vault.

---
## Package Details

| Name | Description | Notes |
| --- | --- | --- |
| main | Entry point. Parses the `-vaultPath`, `-templatesPath` and `-notesPath` flags, then runs `vault.ParseVault` followed by `vault.CleanVault`. | Prints to stderr and exits 1 on any error. |
| types | Holds `Config`, the struct of filesystem paths a run operates on. | Kept in its own package so `vault` and `main` can share it without an import cycle. |
| util | Filesystem helpers: `ReadDirFiles`, `ReadFile` and `WriteFiles` (writes note names back out as `[[wikilinks]]`). | Nothing Obsidian-specific lives here. |
| vault | The vault model and the checks that run against it. `vault.go` interns notes into an ID-indexed graph and writes the report; `note.go` holds the per-note checks (`UsesTemplate`, `ContainsDangling`, `IsOrphan`); `markdown.go` parses wikilinks and YAML frontmatter. | The only package with a third-party dependency (`gopkg.in/yaml.v3`). |


## Requirements

- Go 1.26.7 or newer (version is written in `go.mod`).
- Dependency, [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) v3.0.1, used to parse note frontmatter. `go build` fetches it.

## Usage

Build the binary, then pass the path to your vault:

```sh
go build -o go-obsidian-helper .

./go-obsidian-helper \
  -vaultPath "$HOME/Documents/MyVault" \
  -templatesPath "$HOME/Documents/MyVault/Templates" \
  -notesPath "$HOME/Documents/MyVault/Notes"
```

Or run it without building first:

```sh
go run . -vaultPath ... -templatesPath ... -notesPath ...
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-vaultPath` | `/` | Root of the vault. The report is written here. |
| `-templatesPath` | `/` | Directory of template notes. Their keys define what a well formatted note looks like. |
| `-notesPath` | `/` | Directory of notes to check. |

The three paths are independent, so `-notesPath` can be a single subfolder of a
larger vault while the report still lands at the vault root.

Each directory is read one level deep.

## Output

A run writes a single report to `!! CLEANFILE.md` at the root of
`-vaultPath`. **The file is truncated on every run**, so it is a snapshot of
the last run.

Every line is written as a wikilink so the report is browsable from inside
Obsidian.

```markdown
[[-----BADFORMAT-----]]
[[github.md]]
[[Untitled 3.md]]


[[------DANGLING-----]]
[[gopher.md]]


[[------ORPHANS------]]
[[golang.md]]
```

| Section | Holds |
| --- | --- |
| `BADFORMAT` | Notes that don't carry every key of at least one template — includes notes with no frontmatter at all. |
| `DANGLING` | Notes containing a `[[link]]` that resolves to no file under `-notesPath`. |
| `ORPHANS` | Notes with no outgoing links. |

A note can appear in more than one section. Notes that fail to open are skipped
with a message on stdout and left out of the report entirely.

## References
1 [Google Go Style Guide](https://google.github.io/styleguide/go/guide)  
2 My internship cohost  
3 [Golang and DevOps: How Go Powers Modern CI/CD Pipelines & Infrastructure Automation in 2026](https://newagesysit.com/blog/golang-and-devops-how-go-powers-modern-ci-cd-pipelines-infrastructure-automation/)  