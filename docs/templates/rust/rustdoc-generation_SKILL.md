# Skill: Rustdoc Generation
**Domain:** Code Documentation
**Scope:** Automated Rust documentation tagging.

## Execution Parameters
* Generate `///` block comments for all structs, enums, traits, and public functions.
* Generate `//!` block comments at the top of all module (`mod`) and crate root files.
* Ensure documentation includes `# Arguments`, `# Returns`, and `# Examples` blocks where applicable.
* Maintain syntactical compatibility for automated extraction and rendering via `cargo doc`.